package middleware

import (
	"strings"
	"testing"
)

// Default JWKS harus path Supabase yang benar (F11 langkah 24):
// <supabase>/auth/v1/.well-known/jwks.json. Path lama tanpa /auth/v1 → 404.
func TestNewVerifier_DefaultJWKSPath(t *testing.T) {
	v := NewVerifier("https://abc.supabase.co", "")
	if !strings.HasSuffix(v.url, "/auth/v1/.well-known/jwks.json") {
		t.Fatalf("fallback JWKS harus berakhir /auth/v1/.well-known/jwks.json, dapat %q", v.url)
	}
	if v.url != "https://abc.supabase.co/auth/v1/.well-known/jwks.json" {
		t.Fatalf("URL fallback salah: %q", v.url)
	}
	if v.issuer != "https://abc.supabase.co/auth/v1" {
		t.Fatalf("issuer salah: %q", v.issuer)
	}
}

// trailing slash di supabaseURL tidak menggandakan /auth/v1.
func TestNewVerifier_DefaultJWKSTrailingSlash(t *testing.T) {
	v := NewVerifier("https://abc.supabase.co/", "")
	if v.url != "https://abc.supabase.co/auth/v1/.well-known/jwks.json" {
		t.Fatalf("URL fallback salah: %q", v.url)
	}
}

// SUPABASE_JWKS_URL eksplisit tidak boleh diubah sama sekali.
func TestNewVerifier_JWKSURL_EksplisitDipakai(t *testing.T) {
	const want = "https://self-host/jwks/custom.json"
	v := NewVerifier("https://abc.supabase.co", want)
	if v.url != want {
		t.Fatalf("JWKS eksplisit harus dipakai apa adanya, dapat %q", v.url)
	}
}
