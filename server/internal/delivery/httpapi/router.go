// Route Fase 5: net/http stdlib ServeMux (docs/plan/structure.md "Router").
// Semua endpoint openapi.md bagian user terdaftar. Endpoint admin F7.
package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"paluketok/server/internal/delivery/middleware"
	"paluketok/server/internal/entity"
)

// requestLogger logger dasar server (JSON di prod, teks di dev).
var requestLogger = func() *slog.Logger {
	env := os.Getenv("APP_ENV")
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}()

// RouterOptions adalah dependensi router.
type RouterOptions struct {
	Handlers    *Handlers
	Verifier    *middleware.Verifier // nil di test webhook murni
	AllowOrigin []string             // CORS origin; kosong = dev (buka)
	// ActorFix dipakai TEST saja: pengganti JWT untuk httptest (actor tetap).
	ActorFix entity.Actor
}

// NewRouter merangkai mux + middleware berurutan (docs/plan/workflow.md 19):
// Recover -> RequestID -> CORS -> BodyLimit -> (JWT per route) -> handler.
func NewRouter(o RouterOptions) http.Handler {
	mux := http.NewServeMux()

	healthz := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}

	// health tanpa auth (workflow.md 7).
	mux.HandleFunc("GET /healthz", healthz)

	// Webhook Midtrans: TANPA JWT. Keaslian dibuktikan signature SHA512 di body
	// (lihat webhook.go); tidak ada segmen rahasia di URL.
	mux.HandleFunc("POST /v1/webhooks/midtrans", o.Handlers.WebhookMidtrans)

	// Grup JWT: semua endpoint user-scoped.
	authed := http.NewServeMux()
	authed.HandleFunc("GET /v1/me", o.Handlers.Me)
	authed.HandleFunc("GET /v1/projects", o.Handlers.ListProjects)
	authed.HandleFunc("POST /v1/projects", o.Handlers.CreateProject)
	authed.HandleFunc("GET /v1/projects/{id}", o.Handlers.GetProject)
	authed.HandleFunc("GET /v1/projects/{id}/artifacts", o.Handlers.ListProjectArtifacts)
	// Chat konteks proyek (Fase 4). Ikut jatah userRate 60/menit: runsRate hanya
	// menangkap POST .../runs, jadi tidak perlu rate limit tambahan di sini.
	authed.HandleFunc("POST /v1/projects/{id}/messages", o.Handlers.SendMessage)
	authed.HandleFunc("GET /v1/projects/{id}/messages", o.Handlers.ListMessages)
	authed.HandleFunc("PATCH /v1/projects/{id}", o.Handlers.UpdateProject)
	authed.HandleFunc("DELETE /v1/projects/{id}", o.Handlers.ArchiveProject)
	authed.HandleFunc("POST /v1/projects/{id}/runs", o.Handlers.CreateRun)
	// Usulan skema tahap untuk layar konfirmasi (dinamis per proyek).
	authed.HandleFunc("GET /v1/projects/{id}/stages", o.Handlers.PreviewStages)
	authed.HandleFunc("GET /v1/runs/{id}", o.Handlers.GetRun)
	authed.HandleFunc("GET /v1/runs/{id}/stages", o.Handlers.RunStages)
	authed.HandleFunc("GET /v1/runs/{id}/artifacts", o.Handlers.ListRunArtifacts)
	authed.HandleFunc("GET /v1/artifacts/{id}", o.Handlers.GetArtifact)
	authed.HandleFunc("POST /v1/artifacts/{id}/regenerate", o.Handlers.RegenerateArtifact)
	authed.HandleFunc("GET /v1/decisions", o.Handlers.ListDecisions)
	authed.HandleFunc("POST /v1/orders", o.Handlers.CreateOrder)
	authed.HandleFunc("GET /v1/orders/{id}", o.Handlers.GetOrder)

	// Grup admin: is_admin dari profiles, 403 selain itu (openapi.md bagian admin).
	admin := http.NewServeMux()
	admin.HandleFunc("GET /v1/admin/projects", o.Handlers.AdminListProjects)
	admin.HandleFunc("GET /v1/admin/runs", o.Handlers.AdminListRuns)
	admin.HandleFunc("PATCH /v1/admin/users/{id}", o.Handlers.AdminPatchUser)
	admin.HandleFunc("POST /v1/admin/users/{id}/credits", o.Handlers.AdminGrantCredits)
	admin.HandleFunc("GET /v1/admin/decisions", o.Handlers.AdminListDecisions)
	admin.HandleFunc("GET /v1/admin/settings/ai", o.Handlers.AdminListAISettings)
	admin.HandleFunc("PUT /v1/admin/settings/ai/{slot}", o.Handlers.AdminSetAISetting)

	// Rate limit (security.md bagian 8): key per user dari Actor (fallback IP).
	// POST /runs 5/menit (jatah sendiri, keep), endpoint user lain 60/menit.
	// Urutan: RateLimit(5) terluar, lalu RateLimit(60) — request yang bukan
	// POST /runs lolos jatah 5 tanpa dihitung.
	userRate := middleware.RateLimit(60, time.Minute)
	runsRate := middleware.RateLimitScoped(5, time.Minute, func(r *http.Request) bool {
		return r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/runs")
	})
	adminRate := middleware.RateLimit(30, time.Minute)

	if o.Verifier != nil {
		mux.Handle("/v1/admin/", middleware.JWT(o.Verifier)(middleware.Admin(adminRate(admin))))
		mux.Handle("/v1/", middleware.JWT(o.Verifier)(userRate(runsRate(authed))))
	} else if o.ActorFix.UserID != "" {
		// Hanya test: actor tetap menggantikan JWT.
		mux.Handle("/v1/admin/", middleware.WithActorFix(o.ActorFix)(middleware.Admin(adminRate(admin))))
		mux.Handle("/v1/", middleware.WithActorFix(o.ActorFix)(userRate(runsRate(authed))))
	} else {
		// Test tanpa JWT (webhook murni): actor kosong.
		mux.Handle("/v1/", authed)
	}

	h := middleware.Recover(requestLogger)(mux)
	h = middleware.RequestID(requestLogger)(h)
	h = middleware.CORS(o.AllowOrigin)(h)
	h = middleware.BodyLimit(1 << 20)(h) // 1 MB default; webhook 16 KB di handler
	return h
}
