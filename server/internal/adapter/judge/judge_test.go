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

// TestDecide_ModelDariSlot: "judge" adalah peran penilai, bukan produk. Model
// yang dikirim ke penyedia harus ikut nilai slot, bukan konstanta "jev-latest".
// Kalau ini rusak, admin ganti penyedia penilai tapi permintaan tetap memakai
// model lama, dan panggilannya gagal tanpa sebab yang jelas di panel.
func TestDecide_ModelDariSlot(t *testing.T) {
	var dikirim any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		dikirim = body["model"]
		w.Write([]byte(`{"id":"d","model":"x","answers":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	c.Model = "penilai-baru-1"
	if _, err := c.Decide(context.Background(), port.JudgeRequest{}); err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dikirim != "penilai-baru-1" {
		t.Fatalf("model dari slot tidak dipakai, terkirim: %v", dikirim)
	}
}

// TestDecide_ModelKosongJatuhKeDefault: slot lama tanpa model harus tetap jalan.
func TestDecide_ModelKosongJatuhKeDefault(t *testing.T) {
	var dikirim any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		dikirim = body["model"]
		w.Write([]byte(`{"id":"d","model":"x","answers":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k") // Model sengaja dibiarkan kosong
	if _, err := c.Decide(context.Background(), port.JudgeRequest{}); err != nil {
		t.Fatalf("decide: %v", err)
	}
	if dikirim != DefaultModel {
		t.Fatalf("model kosong harus jatuh ke %q, terkirim: %v", DefaultModel, dikirim)
	}
}
