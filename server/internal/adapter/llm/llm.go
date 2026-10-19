// Package llm client OpenAI-compatible. Endpoint penuh dari ai_settings.BaseURL.
// Timeout 180 detik, tanpa auto-retry (docs/plan/structure.md, docs/plan/jev_ai.md).
// Kredensial per slot: base_url + api_key + model dari baris ai_settings
// (dibaca penelepon, bukan dari sini).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"paluketok/server/internal/port"
)

// Timeout http.Client LLM (docs/plan/structure.md baris 70).
const Timeout = 180 * time.Second

// Config satu slot dari ai_settings (erd.dbml 158-170).
type Config struct {
	BaseURL string // endpoint PENUH, mis. https://gw.example.com/v1/chat/completions
	APIKey  string
	Model   string
}

// errLLM error HTTP dari provider (bukan network).
type errLLM struct {
	Status int
	Body   string
}

func (e *errLLM) Error() string { return fmt.Sprintf("llm: http %d: %s", e.Status, e.Body) }

// IsUnavailable true bila provider menolak (auth/quota/route) -> 503 di usecase.
func IsUnavailable(err error) bool {
	var e *errLLM
	return errors.As(err, &e) && (e.Status == http.StatusUnauthorized ||
		e.Status == http.StatusForbidden || e.Status == http.StatusNotFound ||
		e.Status == http.StatusTooManyRequests || e.Status >= 500)
}

type Client struct{ http *http.Client }

// New client LLM. CheckRedirect menolak redirect lintas host
// (docs/plan/security.md tabel 72, checklist 13 butir 5): base_url dicuri lewat
// redirect open-redirect tidak mungkin.
func New() *Client {
	return &Client{http: &http.Client{
		Timeout: Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				return fmt.Errorf("llm: redirect lintas host ditolak: %s", req.URL.Host)
			}
			if len(via) >= 3 {
				return fmt.Errorf("llm: terlalu banyak redirect")
			}
			return nil
		},
	}}
}

// Complete POST ke cfg.BaseURL apa adanya. Endpoint BUKAN dari klien: base_url
// sudah memuat path lengkapnya (dijaga ValidBaseURLEndpoint).
// hanya nilai Config dari baris ai_settings pilihan server.
func (c *Client) Complete(ctx context.Context, cfg Config, req port.LLMRequest) (*port.LLMResponse, error) {
	msgs := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{
		"model":       cfg.Model,
		"messages":    msgs,
		"temperature": req.Temperature,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(cfg.BaseURL, "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respRaw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, &errLLM{Status: resp.StatusCode, Body: string(respRaw)}
	}
	var dr struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respRaw, &dr); err != nil {
		return nil, err
	}
	if len(dr.Choices) == 0 {
		return nil, fmt.Errorf("llm: tanpa choices: %s", string(respRaw))
	}
	return &port.LLMResponse{
		Content: dr.Choices[0].Message.Content,
		Model:   dr.Model,
		Usage: port.LLMUsage{
			InputTokens:  dr.Usage.PromptTokens,
			OutputTokens: dr.Usage.CompletionTokens,
		},
	}, nil
}
