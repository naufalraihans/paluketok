// Verifikasi JWT Supabase via JWKS (docs/plan/security.md bagian 2).
// Allowlist alg RS256/ES256, kid matching + refresh berbatas, cek iss/aud/exp.
// Tanpa SUPABASE_JWT_SECRET (plan melarang; verifikasi asimetris).
package middleware

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"paluketok/server/internal/entity"
)

// typeKey adalah tipe kunci context untuk Actor.
type typeKey struct{}

// ActorFrom membaca Actor dari context (terpasang middleware JWT).
func ActorFrom(ctx context.Context) (entity.Actor, bool) {
	a, ok := ctx.Value(typeKey{}).(entity.Actor)
	return a, ok
}

// jwksKey adalah satu kunci publik dari JWKS.
type jwksKey struct {
	alg string // "RS256" | "ES256"
	// RSA
	n *big.Int
	e int
	// EC P-256
	ec   elliptic.Curve
	x, y *big.Int
}

type jwksCache struct {
	mu          sync.Mutex
	keys        map[string]jwksKey
	lastAttempt time.Time
	ok          bool
}

// Verifier memverifikasi JWT Supabase. kid tak dikenal memicu refresh JWKS
// dengan batas frekuensi 30 detik (security.md bagian 2).
type Verifier struct {
	cache  *jwksCache
	url    string
	aud    string
	issuer string
	client *http.Client
	clock  func() time.Time
	// IsAdmin baca profiles.is_admin untuk UserID (composition root isi dari
	// repo; security.md bagian 2-3: isAdmin dari DB, bukan claim). Nil di test.
	IsAdmin func(ctx context.Context, userID entity.ID) (bool, error)
}

// NewVerifier membuat verifier. jwksURL kosong = <supabaseURL>/auth/v1/.well-known/jwks.json
// (path JWKS Supabase; tanpa /auth/v1 → 404).
func NewVerifier(supabaseURL, jwksURL string) *Verifier {
	if jwksURL == "" {
		u, err := url.Parse(supabaseURL)
		if err == nil && u.Host != "" {
			jwksURL = u.Scheme + "://" + u.Host + "/auth/v1/.well-known/jwks.json"
		}
	}
	return &Verifier{
		cache:  &jwksCache{},
		url:    jwksURL,
		aud:    "authenticated",
		issuer: strings.TrimRight(supabaseURL, "/") + "/auth/v1",
		client: &http.Client{Timeout: 10 * time.Second},
		clock:  time.Now,
	}
}

// claims adalah klaim JWT yang dipakai server (security.md bagian 2).
type claims struct {
	Iss string `json:"iss"`
	Aud any    `json:"aud"`
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

// Verify memeriksa token dan mengembalikan Actor (tanpa isAdmin — isAdmin
// dibaca dari DB di handler, bukan dari claim, security.md bagian 2-3).
func (v *Verifier) Verify(ctx context.Context, token string) (entity.Actor, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return entity.Actor{}, errors.New("jwt: format salah")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return entity.Actor{}, errors.New("jwt: header rusak")
	}
	var hdr struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return entity.Actor{}, errors.New("jwt: header rusak")
	}
	// Allowlist algoritma: alg none dan lainnya ditolak (security.md bagian 2).
	if hdr.Alg != "RS256" && hdr.Alg != "ES256" {
		return entity.Actor{}, fmt.Errorf("jwt: alg %q ditolak", hdr.Alg)
	}
	if hdr.Kid == "" {
		return entity.Actor{}, errors.New("jwt: tanpa kid")
	}

	key, ok := v.cache.keys[hdr.Kid]
	if !ok {
		if err := v.refresh(ctx); err != nil {
			return entity.Actor{}, err
		}
		key, ok = v.cache.keys[hdr.Kid]
		if !ok {
			return entity.Actor{}, errors.New("jwt: kid tidak dikenal")
		}
	}
	if key.alg != hdr.Alg {
		return entity.Actor{}, errors.New("jwt: alg tidak cocok dengan JWKS")
	}

	signing := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return entity.Actor{}, errors.New("jwt: signature rusak")
	}
	sum := sha256.Sum256([]byte(signing))
	switch key.alg {
	case "RS256":
		pk := &rsa.PublicKey{N: key.n, E: key.e}
		if err := rsa.VerifyPKCS1v15(pk, crypto.SHA256, sum[:], sig); err != nil {
			return entity.Actor{}, errors.New("jwt: signature salah")
		}
	case "ES256":
		if len(sig) != 64 {
			return entity.Actor{}, errors.New("jwt: signature rusak")
		}
		var r, s big.Int
		r.SetBytes(sig[:32])
		s.SetBytes(sig[32:])
		pk := &ecdsa.PublicKey{Curve: key.ec, X: key.x, Y: key.y}
		if !ecdsa.Verify(pk, sum[:], &r, &s) {
			return entity.Actor{}, errors.New("jwt: signature salah")
		}
	}

	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return entity.Actor{}, errors.New("jwt: payload rusak")
	}
	var cl claims
	if err := json.Unmarshal(cb, &cl); err != nil {
		return entity.Actor{}, errors.New("jwt: payload rusak")
	}
	if cl.Iss != v.issuer {
		return entity.Actor{}, errors.New("jwt: iss salah")
	}
	if !audMatch(cl.Aud, v.aud) {
		return entity.Actor{}, errors.New("jwt: aud salah")
	}
	if cl.Exp <= v.clock().Unix() {
		return entity.Actor{}, errors.New("jwt: token kedaluwarsa")
	}
	if cl.Sub == "" {
		return entity.Actor{}, errors.New("jwt: tanpa sub")
	}
	actor := entity.Actor{UserID: entity.ID(cl.Sub)}
	// is_admin dibaca dari DB, bukan claim (security.md bagian 2-3). Gagal
	// baca = tetap non-admin (fail-closed), bukan error 401.
	if v.IsAdmin != nil {
		if ok, err := v.IsAdmin(ctx, actor.UserID); err == nil {
			actor.IsAdmin = ok
		}
	}
	return actor, nil
}

func audMatch(aud any, want string) bool {
	switch a := aud.(type) {
	case string:
		return a == want
	case []any:
		for _, x := range a {
			if s, ok := x.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

// refresh mengambil ulang JWKS dengan batas frekuensi 30 detik.
func (v *Verifier) refresh(ctx context.Context) error {
	v.cache.mu.Lock()
	defer v.cache.mu.Unlock()
	if v.cache.ok && v.clock().Sub(v.cache.lastAttempt) < 30*time.Second {
		return errors.New("jwks: kid tetap tak dikenal")
	}
	v.cache.lastAttempt = v.clock()
	keys, err := fetchJWKS(ctx, v.client, v.url)
	if err != nil {
		return err
	}
	v.cache.keys = keys
	v.cache.lastAttempt = v.clock()
	v.cache.ok = true
	return nil
}

// fetchJWKS mengambil kunci publik dari JWKS URL.
func fetchJWKS(ctx context.Context, client *http.Client, jwksURL string) (map[string]jwksKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks: http %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	keys := make(map[string]jwksKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Alg != "" && k.Alg != "RS256" && k.Alg != "ES256" {
			continue
		}
		switch k.Kty {
		case "RSA":
			n, e1 := base64.RawURLEncoding.DecodeString(k.N)
			eb, e2 := base64.RawURLEncoding.DecodeString(k.E)
			if e1 != nil || e2 != nil || len(eb) > 4 || len(n) == 0 {
				continue
			}
			ei := 0
			for _, b := range eb {
				ei = ei<<8 | int(b)
			}
			keys[k.Kid] = jwksKey{alg: "RS256", n: new(big.Int).SetBytes(n), e: ei}
		case "EC":
			crv, ok := map[string]elliptic.Curve{"P-256": elliptic.P256()}[k.Crv]
			if !ok {
				continue
			}
			xb, e1 := base64.RawURLEncoding.DecodeString(k.X)
			yb, e2 := base64.RawURLEncoding.DecodeString(k.Y)
			if e1 != nil || e2 != nil {
				continue
			}
			keys[k.Kid] = jwksKey{alg: "ES256", ec: crv, x: new(big.Int).SetBytes(xb), y: new(big.Int).SetBytes(yb)}
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("jwks: tanpa kunci yang bisa dipakai")
	}
	return keys, nil
}
