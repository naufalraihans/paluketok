// Harness E2E langkah 29 (bagian 2): mock LLM + JEV, verifier RS256 + JWKS.
// Mock LLM memenuhi worker.LLM (port.LLM): Complete(ctx, req) tanpa Config.
package httpapi

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"paluketok/server/internal/delivery/middleware"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// mockLLM membaca nama tahap dari pesan sistem lalu menjawab konten sah.
type mockLLM struct {
	mu       sync.Mutex
	calls    []entity.Stage
	failOnce map[entity.Stage]bool // sekali gagal validasi (uji retry)
	broken   bool
}

func (m *mockLLM) Complete(_ context.Context, req port.LLMRequest) (*port.LLMResponse, error) {
	stage := stageFromPrompt(req)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.broken {
		return &port.LLMResponse{Content: "rusak", Model: "mock"}, nil
	}
	if m.failOnce[stage] {
		delete(m.failOnce, stage)
		return &port.LLMResponse{Content: "draft rusak tanpa format", Model: "mock"}, nil
	}
	m.calls = append(m.calls, stage)
	return &port.LLMResponse{Content: contentFor(stage, e2ePaths), Model: "mock",
		Usage: port.LLMUsage{InputTokens: 10, OutputTokens: 10}}, nil
}

// stageFromPrompt mencari nama tahap pada pesan sistem ("tahap \"prd\"").
func stageFromPrompt(req port.LLMRequest) entity.Stage {
	for _, m := range req.Messages {
		if m.Role != "system" {
			continue
		}
		for _, s := range entity.Stages {
			if strings.Contains(m.Content, `"`+string(s)+`"`) {
				return s
			}
		}
	}
	return entity.StagePRD
}

// e2ePaths path OpenAPI pada E2E.
var e2ePaths = []string{"/v1/projects", "/v1/runs"}

// contentFor draft per tahap yang lolos worker.validate.
func contentFor(stage entity.Stage, paths []string) string {
	switch stage {
	case entity.StagePRD:
		return "# PRD\n\n## Tujuan\nBangun tumbuh.\n\n## Scope\nSatu fitur.\n\n## Acceptance criteria\nLulus.\n\n## Di luar scope\nLainnya.\n"
	case entity.StageERD:
		return "Table users {\n  id uuid [pk]\n}\nTable projects {\n  id uuid [pk]\n  user_id uuid\n}\nRef: projects.user_id > users.id\n"
	case entity.StageOpenAPI:
		var sb strings.Builder
		sb.WriteString(`{"openapi":"3.1.0","paths":{`)
		for i, p := range paths {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(`"` + p + `":{"get":{"responses":{"200":{"description":"ok"}}}}`)
		}
		sb.WriteString(`}}`)
		return sb.String()
	case entity.StageRoles:
		var sb strings.Builder
		sb.WriteString("| Path | admin | user |\n|---|---|---|\n")
		for _, p := range paths {
			sb.WriteString("| " + p + " | ya | ya |\n")
		}
		return sb.String()
	case entity.StageSecurity:
		return "## Trust boundary\n\nBrowser -> API -> DB.\n\n## Validasi input\n\nSemua input divalidasi.\n\n## Checklist\n\n- [x] auth\n"
	default:
		return "struktur:\nweb/\napi/\n"
	}
}

// snapshot salinan daftar panggilan LLM (thread-safe).
func (m *mockLLM) snapshot() []entity.Stage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]entity.Stage(nil), m.calls...)
}

// --- JEV mock: HTTP server menjawab EXECUTE (confidence 0.95) ---

func jevMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			State     any            `json:"state"`
			Questions map[string]any `json:"questions"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		answers := map[string]any{}
		for qid := range body.Questions {
			answers[qid] = map[string]any{"choice": "EXECUTE", "confidence": 0.95,
				"probabilities": map[string]float64{"EXECUTE": 0.95}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "jev-mock-1", "model": "systemone",
			"state": body.State, "criteria": map[string]any{}, "answers": answers})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- JWT RS256 + JWKS mock ---

type jwtFixture struct {
	key   *rsa.PrivateKey
	kid   string
	verif *middleware.Verifier
}

func newJWTFix(t *testing.T) *jwtFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fx := &jwtFixture{key: key, kid: "kid-e2e"}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{fx.jwk()}})
	}))
	t.Cleanup(jwks.Close)
	fx.verif = middleware.NewVerifier("https://e2e.supabase.co", jwks.URL+"/jwks.json")
	fx.verif.IsAdmin = func(_ context.Context, id entity.ID) (bool, error) {
		return id == uidE2EAdmin, nil
	}
	return fx
}

func (f *jwtFixture) jwk() map[string]any {
	n := base64.RawURLEncoding.EncodeToString(f.key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.PublicKey.E)).Bytes())
	return map[string]any{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": f.kid, "n": n, "e": e}
}

// token membuat JWT RS256 sah untuk sub, exp 2027-01-01.
func (f *jwtFixture) token(t *testing.T, sub string) string {
	t.Helper()
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT","kid":"` + f.kid + `"}`))
	pl, _ := json.Marshal(map[string]any{"iss": "https://e2e.supabase.co/auth/v1",
		"aud": "authenticated", "sub": sub, "exp": 1798761600})
	payload := base64.RawURLEncoding.EncodeToString(pl)
	sig := signRS256Bytes(f.key, hdr+"."+payload)
	return hdr + "." + payload + "." + sig
}

const uidE2E = "0197a23e-7777-7000-8000-00000000000a"
const uidE2EAdmin = "0197a23e-7777-7000-8000-0000000000ad"

// signRS256Bytes tanda tangan RS256 (base64url) atas "hdr.payload".
func signRS256Bytes(key *rsa.PrivateKey, signing string) string {
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(sig)
}
