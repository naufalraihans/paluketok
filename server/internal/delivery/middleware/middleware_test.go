package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// CORS preflight harus mengizinkan PUT (F11 langkah 23): tab AI Settings
// admin simpan slot lewat PUT /v1/admin/settings/ai/{slot}; tanpa PUT di
// Access-Control-Allow-Methods browser memblokir preflight.
func TestCORS_PreflightPUT(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler tidak boleh terpanggil untuk preflight")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/v1/admin/settings/ai/prd", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight ingin 204, dapat %d", rec.Code)
	}
	methods := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "PUT") {
		t.Fatalf("Access-Control-Allow-Methods harus memuat PUT, dapat %q", methods)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("Allow-Origin salah: %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

// Request biasa dengan Origin juga membawa header CORS.
func TestCORS_RequestBiasaDenganOrigin(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "PUT") {
		t.Fatal("header CORS request biasa juga harus memuat PUT")
	}
}
