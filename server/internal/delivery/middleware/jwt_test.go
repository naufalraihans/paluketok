// Test JWT middleware: JWKS server palsu (RSA asli), token asli/salah,
// expired, aud salah, alg none, kid asing, refresh.
package middleware

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixedClock struct{ t time.Time }

func (f *fixedClock) Now() time.Time { return f.t }

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func jwkFor(t *testing.T, kid string, key *rsa.PublicKey) map[string]any {
	t.Helper()
	return map[string]any{
		"kty": "RSA", "kid": kid, "alg": "RS256",
		"n": b64(key.N.Bytes()), "e": b64(bigEndianBytes(key.E)),
	}
}

func bigEndianBytes(n int) []byte {
	if n == 0 {
		return []byte{0}
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte(n & 0xff)}, out...)
		n >>= 8
	}
	return out
}

func signRS256(t *testing.T, key *rsa.PrivateKey, header, payload string) string {
	t.Helper()
	signing := b64([]byte(header)) + "." + b64([]byte(payload))
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + b64(sig)
}

func hs(kid, alg string) string {
	hb, _ := json.Marshal(map[string]string{"alg": alg, "kid": kid})
	return string(hb)
}

// newTestVerifier membuat Verifier dengan JWKS URL server palsu.
func newTestVerifier(t *testing.T, kid string, key *rsa.PublicKey, supabaseURLOverride string) *Verifier {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "jwks.json") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwkFor(t, kid, key)}})
	})
	jwksSrv := httptest.NewServer(mux)
	t.Cleanup(jwksSrv.Close)
	base := "https://abc.supabase.co"
	if supabaseURLOverride != "" {
		base = supabaseURLOverride
	}
	v := NewVerifier(base, jwksSrv.URL+"/.well-known/jwks.json")
	fc := &fixedClock{t: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	v.clock = fc.Now
	return v
}

const testSub = "0197a23e-1111-7000-8000-00000000000a"

func TestJWT_OK(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newTestVerifier(t, "kid-1", &key.PublicKey, "")
	payload, _ := json.Marshal(map[string]any{
		"iss": "https://abc.supabase.co/auth/v1", "aud": "authenticated",
		"sub": testSub, "exp": 1798761600, // 2027-01-01 > clock 2026-10-13
	})
	actor, err := v.Verify(context.Background(), signRS256(t, key, hs("kid-1", "RS256"), string(payload)))
	if err != nil {
		t.Fatalf("harus sah: %v", err)
	}
	if string(actor.UserID) != testSub {
		t.Fatalf("sub salah: %q", actor.UserID)
	}
}

// exp kedaluwarsa, aud salah, kid asing, alg none, signature rusak: semuanya
// ditolak (Verify error).
func TestJWT_TolakSemuaKasusBuruk(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	enc := func(m map[string]any) string {
		b, _ := json.Marshal(m)
		return string(b)
	}
	cases := []struct {
		name    string
		kid     string
		alg     string
		payload string
		key     *rsa.PrivateKey
	}{
		{"expired", "kid-1", "RS256", enc(map[string]any{"iss": "https://abc.supabase.co/auth/v1", "aud": "authenticated", "sub": testSub, "exp": 1000}), key},
		{"aud_salah", "kid-1", "RS256", enc(map[string]any{"iss": "https://abc.supabase.co/auth/v1", "aud": "anon", "sub": testSub, "exp": 9999999999}), key},
		{"iss_salah", "kid-1", "RS256", enc(map[string]any{"iss": "https://evil.example/auth/v1", "aud": "authenticated", "sub": testSub, "exp": 9999999999}), key},
		{"alg_none", "kid-1", "none", enc(map[string]any{"iss": "https://abc.supabase.co/auth/v1", "aud": "authenticated", "sub": testSub, "exp": 9999999999}), key},
		{"kid_asing", "kid-lain", "RS256", enc(map[string]any{"iss": "https://abc.supabase.co/auth/v1", "aud": "authenticated", "sub": testSub, "exp": 9999999999}), key},
		{"tanda_tangan_kunci_lain", "kid-1", "RS256", enc(map[string]any{"iss": "https://abc.supabase.co/auth/v1", "aud": "authenticated", "sub": testSub, "exp": 9999999999}), mustKey(t)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := newTestVerifier(t, "kid-1", &key.PublicKey, "")
			_, err := v.Verify(context.Background(), signRS256(t, c.key, hs(c.kid, c.alg), c.payload))
			if err == nil {
				t.Fatalf("%s harus ditolak", c.name)
			}
		})
	}
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	return k
}

// header Bearer hilang di handler = 401 (middleware level).
func TestJWT_HandlerTanpaBearer(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newTestVerifier(t, "kid-1", &key.PublicKey, "")
	srv := httptest.NewServer(JWT(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a, _ := ActorFrom(r.Context())
		_, _ = w.Write([]byte(string(a.UserID)))
	})))
	defer srv.Close()
	resp, _ := http.Get(srv.URL)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tanpa bearer: ingin 401, dapat %d", resp.StatusCode)
	}
}
