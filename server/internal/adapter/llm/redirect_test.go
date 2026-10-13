// Bukti checklist §13 butir 5 (security.md): client LLM menolak redirect
// lintas host, sehingga base_url sah tidak bisa dicuri ke host lain.
package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"paluketok/server/internal/port"
)

func TestClient_TolakRedirectLintasHost(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"bocor"}}],"usage":{}}`))
	}))
	defer target.Close()

	// Host penyerang yang mengarahkan ke target.
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/v1/chat/completions", http.StatusFound)
	}))
	defer attacker.Close()

	c := New()
	cfg := Config{BaseURL: attacker.URL, APIKey: "k", Model: "m"}
	_, err := c.Complete(context.Background(), cfg, port.LLMRequest{
		Messages: []port.LLMMessage{{Role: "user", Content: "halo"}},
	})
	if err == nil {
		t.Fatal("redirect lintas host harusnya ditolak, dapat nil error")
	}
	if !strings.Contains(err.Error(), "redirect lintas host") {
		t.Fatalf("error tak sesuai: %v", err)
	}
}
