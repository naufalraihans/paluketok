// Package judge client JEV (POST {base} dengan model "jev-latest").
// Timeout 30 detik, TANPA retry, TANPA Idempotency-Key: panggilan bisa sudah
// tercharge (plan/jev_ai.md bagian "How To Use").
package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"paluketok/server/internal/port"
)

// Timeout http.Client JEV (plan/structure.md baris 70).
const Timeout = 30 * time.Second

// Model JEV (plan/jev_ai.md).
const Model = "jev-latest"

// errJudge error HTTP dari gateway JEV.
type errJudge struct {
	Status int
	Body   string
}

func (e *errJudge) Error() string { return fmt.Sprintf("judge: http %d: %s", e.Status, e.Body) }

// IsUnavailable true bila gateway menolak/hangus (auth, 429, 5xx).
func IsUnavailable(err error) bool {
	var e *errJudge
	if !errors.As(err, &e) {
		return false
	}
	return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden ||
		e.Status == http.StatusTooManyRequests || e.Status >= 500
}

// Client JEV. BaseURL tanpa trailing slash, mis.
// https://api.experientiallabs.ai/v1/systemone.
type Client struct {
	BaseURL string // dari konfigurasi server, bukan dari klien
	APIKey  string // JEV_API_KEY, hanya server
	http    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, http: &http.Client{Timeout: Timeout}}
}

// Decide satu panggilan gate. State/questions lewat any: bentuk bebas
// (objek/text/array) sesuai gateway JEV.
func (c *Client) Decide(ctx context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	questions := map[string]any{}
	for _, q := range req.Questions {
		questions[q.ID] = map[string]any{
			"type":         q.Type,
			"instructions": q.Instructions,
			"criteria":     q.Criteria,
		}
	}
	body := map[string]any{
		"model":     Model,
		"state":     req.State,
		"questions": questions,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respRaw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, &errJudge{Status: resp.StatusCode, Body: string(respRaw)}
	}
	var dr struct {
		ID       string `json:"id"`
		Model    string `json:"model"`
		State    any    `json:"state"`
		Criteria any    `json:"criteria"`
		Answers  map[string]struct {
			Choice        string             `json:"choice"`
			Confidence    float64            `json:"confidence"`
			Probabilities map[string]float64 `json:"probabilities"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(respRaw, &dr); err != nil {
		return nil, err
	}
	out := &port.JudgeResponse{
		DecisionID: dr.ID,
		Model:      dr.Model,
		Answers:    map[string]port.JudgeAnswer{},
		State:      dr.State,
		Criteria:   dr.Criteria,
	}
	for id, a := range dr.Answers {
		out.Answers[id] = port.JudgeAnswer{
			QuestionID: id, Choice: a.Choice,
			Confidence: a.Confidence, Probabilities: a.Probabilities,
		}
	}
	return out, nil
}
