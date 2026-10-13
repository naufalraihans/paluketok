package judge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"paluketok/server/internal/port"
)

// Mock server lokal — TIDAK ADA panggil live di test.
func TestDecide_OK(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("body rusak: %v", err)
		}
		w.Write([]byte(`{
		  "id":"dec-123","model":"jev-latest",
		  "state":{"task":"uji"},
		  "answers":{
		    "review":{"choice":"accept","confidence":0.93,"probabilities":{"accept":0.93,"review":0.07}}
		  }
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "jev-key-test")
	resp, err := c.Decide(context.Background(), port.JudgeRequest{
		State: map[string]any{"task": "uji"},
		Questions: []port.JudgeQuestion{{
			ID: "review", Type: "choice", Instructions: "apakah layak?", Criteria: map[string]string{
				"accept": "ya", "review": "perlu tinjau",
			},
		}},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if resp.DecisionID != "dec-123" || resp.Model != "jev-latest" {
		t.Fatalf("id/model salah: %+v", resp)
	}
	a, ok := resp.Answers["review"]
	if !ok || a.Choice != "accept" || a.Confidence < 0.9 {
		t.Fatalf("jawaban salah: %+v", resp.Answers)
	}
	if gotAuth != "Bearer jev-key-test" {
		t.Fatalf("auth salah: %q", gotAuth)
	}
	if gotBody["model"] != "jev-latest" {
		t.Fatalf("model request salah: %v", gotBody["model"])
	}
}

func TestDecide_429Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.Decide(context.Background(), port.JudgeRequest{})
	if !IsUnavailable(err) {
		t.Fatalf("429 harus unavailable, dapat %v", err)
	}
}

func TestTimeout_Konstanta30(t *testing.T) {
	if Timeout != 30*time.Second {
		t.Fatalf("timeout judge harus 30s, dapat %v", Timeout)
	}
}
