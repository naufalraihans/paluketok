package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"paluketok/server/internal/port"
)

// Server palsu OpenAI-compatible lokal: tangkap header/body, balas standar.
func TestComplete_OK(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"uji-model","choices":[{"message":{"content":"halo"}}],
		  "usage":{"prompt_tokens":11,"completion_tokens":7}}`))
	}))
	defer srv.Close()

	c := New()
	// BaseURL kini endpoint PENUH: klien tidak menambah path apa pun.
	resp, err := c.Complete(context.Background(), Config{
		BaseURL: srv.URL + "/v1/chat/completions", APIKey: "sk-test", Model: "uji-model",
	}, port.LLMRequest{
		Slot: "planner",
		Messages: []port.LLMMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hai"},
		},
		MaxTokens: 100, Temperature: 0.2,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Content != "halo" || resp.Model != "uji-model" ||
		resp.Usage.InputTokens != 11 || resp.Usage.OutputTokens != 7 {
		t.Fatalf(" respons salah: %+v", resp)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth header salah: %q", gotAuth)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path salah: %q", gotPath)
	}
}

func TestComplete_401Tersedia(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	c := New()
	_, err := c.Complete(context.Background(), Config{BaseURL: srv.URL, Model: "m"},
		port.LLMRequest{Messages: []port.LLMMessage{{Role: "user", Content: "x"}}})
	if !IsUnavailable(err) {
		t.Fatalf("401 harus dianggap unavailable, dapat %v", err)
	}
}

// Timeout 180 detik tercermin di http.Client; uji dengan context cepat
// supaya test tidak 3 menit.
func TestComplete_TimeoutContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Complete(ctx, Config{BaseURL: srv.URL, Model: "m"},
		port.LLMRequest{Messages: []port.LLMMessage{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("context habis harus error")
	}
}

func TestTimeout_Konstanta180(t *testing.T) {
	if Timeout != 180*time.Second {
		t.Fatalf("timeout LLM harus 180s, dapat %v", Timeout)
	}
	if New().http.Timeout != Timeout {
		t.Fatal("client timeout tidak terpasang")
	}
}
