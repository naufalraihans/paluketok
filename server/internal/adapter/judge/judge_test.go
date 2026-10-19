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

// Mock server lokal - TIDAK ADA panggil live di test.
//
// Bentuk permintaan dan jawaban di sini mengikuti OpenAI Decisions API apa
// adanya: {"input","questions":[{type,name,instructions,choices}]} dan
// answers berupa ARRAY berisi {"name","choice","probabilities":[{value,probability}]}.
// Bentuk lama (state/criteria + answers sebagai map) DITOLAK server, jadi test
// ini yang menjaga agar tidak kembali ke sana.
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
		  "id":"dec-123","model":"gpt-6-luna-decisions",
		  "answers":[
		    {"type":"choice","name":"review","choice":"accept","confidence":0.93,
		     "probabilities":[{"value":"accept","probability":0.93},{"value":"review","probability":0.07}]}
		  ]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "judge-key-test")
	resp, err := c.Decide(context.Background(), port.JudgeRequest{
		Input: map[string]any{"task": "uji"},
		Questions: []port.JudgeQuestion{{
			ID: "review", Type: "choice", Instructions: "apakah layak?",
			Choices: []port.JudgeChoice{
				{Value: "accept", Description: "ya"},
				{Value: "review", Description: "perlu tinjau"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if resp.DecisionID != "dec-123" || resp.Model != "gpt-6-luna-decisions" {
		t.Fatalf("id/model salah: %+v", resp)
	}
	a, ok := resp.Answers["review"]
	if !ok || a.Choice != "accept" || a.Confidence < 0.9 {
		t.Fatalf("jawaban salah: %+v", resp.Answers)
	}
	if a.Probabilities["accept"] < 0.9 {
		t.Fatalf("probabilities tidak terurai: %+v", a.Probabilities)
	}
	if gotAuth != "Bearer judge-key-test" {
		t.Fatalf("auth salah: %q", gotAuth)
	}
	if gotBody["model"] != DefaultModel {
		t.Fatalf("model request salah: %v", gotBody["model"])
	}
}

// TestDecide_BentukBodySesuaiAPI: field "input" dan "questions" ARRAY berisi
// name/choices. Ini inti perbaikan; kalau rusak, server jawab 400 dan tombol
// tes koneksi di panel admin selalu merah tanpa sebab yang jelas.
func TestDecide_BentukBodySesuaiAPI(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"id":"d","model":"m","answers":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.Decide(context.Background(), port.JudgeRequest{
		Input: map[string]any{"brief": "kasir warung"},
		Questions: []port.JudgeQuestion{{
			ID: "q1", Type: "choice", Instructions: "pilih",
			Choices: []port.JudgeChoice{{Value: "ringkas", Description: "empat tahap"}},
		}},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if _, ada := gotBody["state"]; ada {
		t.Fatalf("body masih memakai \"state\"; API minta \"input\": %+v", gotBody)
	}
	if _, ada := gotBody["input"]; !ada {
		t.Fatalf("body tidak punya \"input\": %+v", gotBody)
	}
	qs, ok := gotBody["questions"].([]any)
	if !ok {
		t.Fatalf("\"questions\" harus array, dapat %T: %+v", gotBody["questions"], gotBody["questions"])
	}
	if len(qs) != 1 {
		t.Fatalf("jumlah pertanyaan salah: %d", len(qs))
	}
	q, _ := qs[0].(map[string]any)
	if q["name"] != "q1" || q["type"] != "choice" || q["instructions"] == nil {
		t.Fatalf("pertanyaan tidak punya name/type/instructions: %+v", q)
	}
	if _, ada := q["criteria"]; ada {
		t.Fatalf("pertanyaan masih memakai \"criteria\"; API minta \"choices\": %+v", q)
	}
	ch, ok := q["choices"].([]any)
	if !ok || len(ch) != 1 {
		t.Fatalf("\"choices\" harus array berisi 1: %+v", q["choices"])
	}
	c0, _ := ch[0].(map[string]any)
	if c0["value"] != "ringkas" || c0["description"] == nil {
		t.Fatalf("choice tidak punya value/description: %+v", c0)
	}
}

// TestDecide_InputObjekJadiString: "input" menerima string. Objek harus
// diserialkan, bukan dikirim sebagai objek mentah.
func TestDecide_InputObjekJadiString(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"id":"d","model":"m","answers":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	if _, err := c.Decide(context.Background(), port.JudgeRequest{
		Input: map[string]any{"brief": "kasir"},
	}); err != nil {
		t.Fatalf("decide: %v", err)
	}
	s, ok := gotBody["input"].(string)
	if !ok {
		t.Fatalf("\"input\" harus string, dapat %T", gotBody["input"])
	}
	if !json.Valid([]byte(s)) {
		t.Fatalf("\"input\" bukan JSON: %q", s)
	}
}

// TestDecide_StateDiteruskanKeRespons: API tidak mengembalikan input, jadi
// worker harus tetap menerima salinannya untuk decisions.state.
func TestDecide_StateDiteruskanKeRespons(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"d","model":"m","answers":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	resp, err := c.Decide(context.Background(), port.JudgeRequest{
		Input: map[string]any{"brief": "kasir"},
		Questions: []port.JudgeQuestion{{
			ID: "q1", Choices: []port.JudgeChoice{{Value: "ringkas"}},
		}},
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	m, ok := resp.State.(map[string]any)
	if !ok || m["brief"] != "kasir" {
		t.Fatalf("state tidak diteruskan: %+v", resp.State)
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

// TestDecide_400TidakUnavailable: model/format salah = 400. Itu kesalahan
// konfigurasi admin, bukan gangguan penyedia, jadi TIDAK boleh dianggap
// unavailable: pesannya harus sampai ke panel supaya bisa diperbaiki.
func TestDecide_400TidakUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"The model 'jev-latest' does not serve the OpenAI Decisions API."}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.Decide(context.Background(), port.JudgeRequest{})
	if err == nil {
		t.Fatal("400 harus jadi error")
	}
	if IsUnavailable(err) {
		t.Fatalf("400 bukan unavailable (salah konfigurasi): %v", err)
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
		w.Write([]byte(`{"id":"d","model":"x","answers":[]}`))
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
		w.Write([]byte(`{"id":"d","model":"x","answers":[]}`))
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
