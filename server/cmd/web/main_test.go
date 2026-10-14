// Smoke test composition root (tolakan F5): sistem penuh dirakit seperti
// produksi lewat newServer("../../.env") — TANPA merakit Deps sendiri — lalu server
// nyata ListenAndServe di 127.0.0.1 port acak dan /healthz divalidasi HTTP 200.
// Ini membuktikan panic "IDGenerator wajib diisi" tidak kembali.
package main

import (
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestNewServer_CompositionRoot merakit composition root asli (pool Supabase
// sandbox dari env) lalu menyodok /healthz dan /v1/me dari port nyata.
// Path .env relatif cwd paket (server/cmd/web), jadi "../../.env".
func TestNewServer_CompositionRoot(t *testing.T) {
	if os.Getenv("PALUKETOK_SMOKE") == "" {
		t.Skip("smoke butuh DATABASE_URL sandbox: set PALUKETOK_SMOKE=1")
	}
	const envPath = "../../.env"
	if _, err := os.Stat(envPath); err != nil {
		t.Skipf("smoke: %s tidak ada", envPath)
	}

	srv, err := newServer(envPath)
	if err != nil {
		t.Fatalf("composition root gagal: %v", err)
	}
	defer srv.http.Close()

	// Port 0: aman dari bentrok; alamat nyata diambil setelah listen.
	srv.http.Addr = "127.0.0.1:0"
	ln, err := net.Listen("tcp", srv.http.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.http.Serve(ln) }()

	client := &http.Client{Timeout: 5 * time.Second}
	base := "http://" + ln.Addr().String()

	var code int
	for i := 0; i < 40; i++ {
		resp, err := client.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			code = resp.StatusCode
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if code != http.StatusOK {
		t.Fatalf("/healthz ingin 200, dapat %d", code)
	}

	// Grup authed terpasang sampai JWT: tanpa token 401 dari middleware
	// (bukan panic) — chain utuh dari composition root.
	resp, err := client.Get(base + "/v1/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/v1/me tanpa token ingin 401, dapat %d", resp.StatusCode)
	}
}
