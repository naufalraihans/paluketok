package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"paluketok/server/internal/usecase"
)

// F11 langkah 26: rate limit terpasang di router (security.md bagian 8).
// Skenario dua user di SATU router (JWT mock): A kuras 60/menit, B tetap
// lancar; POST /runs kena jatah 5/menit sendiri; webhook tanpa limit.

const uidRLB = "0197a23e-7777-7000-8000-0000000000bb"

func rlServer(t *testing.T) (*httptest.Server, *jwtFixture) {
	t.Helper()
	fix := newJWTFix(t)
	st := newMemStore()
	st.seedProfile(uidE2E, 1)
	st.seedProfile(uidRLB, 1)
	h := &Handlers{Credits: usecase.NewCreditUsecase(usecase.Deps{Profiles: memProfiles{st}})}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h, Verifier: fix.verif}))
	t.Cleanup(srv.Close)
	return srv, fix
}

func rlDo(t *testing.T, srv *httptest.Server, method, path, token string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 200))
	return res.StatusCode, string(body)
}

// Runs nil: handler panic, Recover balik 500 — cukup sebagai "bukan 429".
func TestRateLimit_429DanJatahPerUser(t *testing.T) {
	srv, fix := rlServer(t)
	tokA := fix.token(t, uidE2E)
	tokB := fix.token(t, uidRLB)

	// POST /runs: 5 pertama lewat limiter (zustand 500 dari Recover), ke-6 = 429.
	for i := 0; i < 5; i++ {
		code, _ := rlDo(t, srv, http.MethodPost, "/v1/projects/p1/runs", tokB)
		if code == http.StatusTooManyRequests {
			t.Fatalf("POST /runs ke-%d tidak boleh 429", i+1)
		}
	}
	code, body := rlDo(t, srv, http.MethodPost, "/v1/projects/p1/runs", tokB)
	if code != http.StatusTooManyRequests {
		t.Fatalf("POST /runs ke-6 ingin 429, dapat %d", code)
	}
	if !strings.Contains(body, "rate_limited") {
		t.Fatalf("body 429 harus code rate_limited: %s", body)
	}

	// A kuras 60/menit di /v1/me; ke-61 = 429.
	for i := 0; i < 60; i++ {
		if c, _ := rlDo(t, srv, http.MethodGet, "/v1/me", tokA); c == http.StatusTooManyRequests {
			t.Fatalf("GET /v1/me ke-%d tidak boleh 429", i+1)
		}
	}
	if c, _ := rlDo(t, srv, http.MethodGet, "/v1/me", tokA); c != http.StatusTooManyRequests {
		t.Fatalf("GET /v1/me ke-61 ingin 429, dapat %d", c)
	}

	// B (router sama): bucket 60-nya masih hidup.
	if c, _ := rlDo(t, srv, http.MethodGet, "/v1/me", tokB); c != http.StatusOK {
		t.Fatalf("GET /v1/me user B ingin 200, dapat %d (jatah per-user bocor)", c)
	}

	// Webhook: tanpa JWT, tidak lewat limiter /v1/ (signature gagal pun tak 429).
	for i := 0; i < 3; i++ {
		if c, _ := rlDo(t, srv, http.MethodPost, "/v1/webhooks/midtrans", ""); c == http.StatusTooManyRequests {
			t.Fatal("webhook tidak boleh kena rate limit")
		}
	}
}
