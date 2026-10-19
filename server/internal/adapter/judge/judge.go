// Package judge client JEV (POST {base} dengan model "jev-latest").
// Timeout 30 detik, TANPA retry, TANPA Idempotency-Key: panggilan bisa sudah
// tercharge (docs/plan/jev_ai.md bagian "How To Use").
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

// DefaultModel dipakai HANYA kalau slot judge tidak menyebut model.
// "judge" adalah peran penilai, bukan produk: penyedia bisa ditukar dari admin
// tanpa deploy ulang, dan nilai ini menjaga data lama tetap jalan.
const DefaultModel = "jev-latest"

// Timeout http.Client penilai (docs/plan/structure.md baris 70).
const Timeout = 30 * time.Second

// ErrHTTP error HTTP dari gateway penilai. Diekspor supaya pemanggil (mis.
// tombol tes koneksi panel admin) bisa meneruskan pesan mentah penyedia.
type ErrHTTP struct {
	Status int
	Body   string
}

func (e *ErrHTTP) Error() string { return fmt.Sprintf("judge: http %d: %s", e.Status, e.Body) }

// IsUnavailable true bila gateway menolak/hangus (auth, 429, 5xx).
func IsUnavailable(err error) bool {
	var e *ErrHTTP
	if !errors.As(err, &e) {
		return false
	}
	return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden ||
		e.Status == http.StatusTooManyRequests || e.Status >= 500
}

// Client penilai. BaseURL tanpa trailing slash, mis.
// https://api.experientiallabs.ai/v1/systemone.
// HTTPClient dipakai semua Client. Diekspor sebagai seam tes: paket lain
// (mis. usecase) perlu mengarahkan panggilan ke server palsu, sedangkan
// ValidBaseURLEndpoint menolak host lokal sehingga URL httptest tidak bisa
// dipakai langsung.
var HTTPClient = &http.Client{Timeout: Timeout}

type Client struct {
	BaseURL string // dari konfigurasi server, bukan dari klien
	APIKey  string // hanya di server
	// Model nama model penilai. Kosong = DefaultModel. Diisi dari slot
	// ai_settings supaya penyedia penilai bisa diganti tanpa deploy ulang.
	Model string
	http  *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, http: HTTPClient}
}

// questionChoices merangkum pilihan tiap pertanyaan untuk kolom decisions.criteria.
func questionChoices(qs []port.JudgeQuestion) any {
	out := map[string]any{}
	for _, q := range qs {
		vals := make([]string, 0, len(q.Choices))
		for _, ch := range q.Choices {
			vals = append(vals, ch.Value)
		}
		out[q.ID] = map[string]any{"options": vals}
	}
	return out
}

// Decide satu panggilan gate. Bentuk body persis OpenAI Decisions API:
// POST {BaseURL} dengan {"model","input","questions":[{type,name,instructions,choices}]}.
// State/questions lewat any: bentuk bebas (objek/text/array).
func (c *Client) Decide(ctx context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	questions := make([]map[string]any, 0, len(req.Questions))
	for _, q := range req.Questions {
		choices := make([]map[string]any, 0, len(q.Choices))
		for _, ch := range q.Choices {
			choices = append(choices, map[string]any{
				"value":       ch.Value,
				"description": ch.Description,
			})
		}
		questions = append(questions, map[string]any{
			"type":         q.Type,
			"name":         q.ID,
			"instructions": q.Instructions,
			"choices":      choices,
		})
	}
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	// "input" menerima objek: dikirim sebagai string JSON supaya server tidak
	// menolak tipe. Bentuk objek dipertahankan karena penilai butuh struktur.
	input := req.Input
	if input != nil {
		if b, err := json.Marshal(input); err == nil {
			input = string(b)
		}
	}
	body := map[string]any{
		"model":     model,
		"input":     input,
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
		return nil, &ErrHTTP{Status: resp.StatusCode, Body: string(respRaw)}
	}
	// Bentuk jawaban OpenAI Decisions API: answers adalah ARRAY, tiap entri
	// punya field "name" (bukan map berkunci nama).
	var dr struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Answers []struct {
			Type          string  `json:"type"`
			Name          string  `json:"name"`
			Choice        string  `json:"choice"`
			Confidence    float64 `json:"confidence"`
			Probabilities []struct {
				Value       string  `json:"value"`
				Probability float64 `json:"probability"`
			} `json:"probabilities"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(respRaw, &dr); err != nil {
		return nil, err
	}
	out := &port.JudgeResponse{
		DecisionID: dr.ID,
		Model:      dr.Model,
		Answers:    map[string]port.JudgeAnswer{},
		// API tidak mengembalikan input/state, jadi diteruskan apa adanya dari
		// permintaan supaya worker tetap bisa mencatat keputusan.
		State:    req.Input,
		Criteria: questionChoices(req.Questions),
	}
	for _, a := range dr.Answers {
		probs := map[string]float64{}
		for _, p := range a.Probabilities {
			probs[p.Value] = p.Probability
		}
		out.Answers[a.Name] = port.JudgeAnswer{
			QuestionID: a.Name, Choice: a.Choice,
			Confidence: a.Confidence, Probabilities: probs,
		}
	}
	return out, nil
}
