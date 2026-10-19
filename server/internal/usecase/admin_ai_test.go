package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"paluketok/server/internal/adapter/judge"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// TestAISetting memakai Model dari FORM, bukan judge.DefaultModel.
//
// Insiden nyata: form admin berisi "gpt-6-luna-decisions", tapi tes koneksi
// menembak model "jev-latest" karena client dibuat tanpa Model. Provider balas
// 400 "model 'jev-latest' does not serve the OpenAI Decisions API" dan panel
// hanya menampilkan "kesalahan server".
//
// ValidBaseURLEndpoint menolak host lokal, jadi URL httptest tidak bisa dipakai
// langsung. Arahkan lewat seam judge.HTTPClient.
func TestTestAISetting_PakaiModelDariForm(t *testing.T) {
	var gotModel, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"answers":[{"type":"choice","name":"koneksi","choice":"ok"}]}`))
	}))
	defer srv.Close()

	prev := judge.HTTPClient
	judge.HTTPClient = &http.Client{Transport: rewriteTo(srv.URL)}
	defer func() { judge.HTTPClient = prev }()

	u := NewAdminUsecase(Deps{}, nil)
	actor := entity.Actor{UserID: "u1", IsAdmin: true}
	ok, _, err := u.TestAISetting(context.Background(), actor, port.AISetting{
		Slot: "judge", BaseURL: "https://api.experientiallabs.ai/v1/decisions",
		Model: "gpt-6-luna-decisions", APIKey: "k",
	})
	if err != nil {
		t.Fatalf("test setting: %v", err)
	}
	if !ok {
		t.Fatal("harus ok")
	}
	if gotModel != "gpt-6-luna-decisions" {
		t.Fatalf("model terkirim %q, harus dari form (bukan DefaultModel)", gotModel)
	}
	// Base URL dipakai apa adanya: tidak ada path tambahan dari klien.
	if gotPath != "/v1/decisions" {
		t.Fatalf("path terkirim %q, harus tepat seperti base_url", gotPath)
	}
}

// Model yang ditolak provider diteruskan apa adanya, tidak disamarkan jadi
// "kesalahan server".
func TestTestAISetting_ErrorProviderDiteruskan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"The model 'x' does not serve the OpenAI Decisions API.","code":"unsupported_capability"}}`))
	}))
	defer srv.Close()

	prev := judge.HTTPClient
	judge.HTTPClient = &http.Client{Transport: rewriteTo(srv.URL)}
	defer func() { judge.HTTPClient = prev }()

	u := NewAdminUsecase(Deps{}, nil)
	actor := entity.Actor{UserID: "u1", IsAdmin: true}
	_, _, err := u.TestAISetting(context.Background(), actor, port.AISetting{
		Slot: "judge", BaseURL: "https://api.experientiallabs.ai/v1/decisions",
		Model: "x", APIKey: "k",
	})
	if err == nil {
		t.Fatal("harus error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "400") || !strings.Contains(msg, "does not serve") {
		t.Fatalf("pesan provider tidak diteruskan: %v", err)
	}
}

// rewriteTo mengalihkan setiap request ke server tes, apa pun host tujuannya.
func rewriteTo(target string) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		u := *r.URL
		u.Scheme, u.Host = "http", strings.TrimPrefix(target, "http://")
		r2 := r.Clone(r.Context())
		r2.URL = &u
		return http.DefaultTransport.RoundTrip(r2)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
