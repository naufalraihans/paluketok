// Rantai middleware Fase 5 (plan/workflow.md 19, plan/security.md):
// Recover -> RequestID/Log -> CORS -> BodyLimit -> JWT -> Admin -> RateLimit.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/pkg/response"
)

// Recover menangkap panic per request: log + 500, proses tidak mati.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", "err", rec, "stack", string(debug.Stack()))
					response.JSON(w, http.StatusInternalServerError, map[string]any{
						"error": response.ErrorBody{Code: response.CodeInternal, Message: "internal"},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID menandai tiap request dengan X-Request-Id dan logger konteks.
func RequestID(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				var b [8]byte
				_, _ = rand.Read(b[:])
				id = hex.EncodeToString(b[:])
			}
			rlog := log.With("req_id", id, "method", r.Method, "path", r.URL.Path)
			start := time.Now()
			r = r.WithContext(context.WithValue(r.Context(), logKey{}, rlog))
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r)
			rlog.Info("request", "dur", time.Since(start).String())
		})
	}
}

type logKey struct{}

// LogFrom mengembalikan logger request (dipakai handler). nil logger aman:
// jatuh ke slog.Default().
func LogFrom(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(logKey{}).(*slog.Logger); ok {
			return l
		}
	}
	return slog.Default()
}

// CORS: allow origin terbatas dari BASE_URL klien (dev = *). F5 sederhana:
// method dan header standar.
func CORS(allowOrigins []string) func(http.Handler) http.Handler {
	allow := map[string]bool{}
	for _, o := range allowOrigins {
		allow[strings.TrimRight(o, "/")] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allow[origin] || len(allow) == 0) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit membatasi ukuran body (security.md: 16 KB webhook, 1 MB default).
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// JWT memverifikasi Bearer token dan menaruh Actor di context.
// Gagal = 401 (security.md bagian 2).
func JWT(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) || strings.TrimSpace(h[len(prefix):]) == "" {
				response.JSON(w, http.StatusUnauthorized, map[string]any{
					"error": response.ErrorBody{Code: response.CodeUnauthorized, Message: "token tidak ada"},
				})
				return
			}
			actor, err := v.Verify(r.Context(), strings.TrimSpace(h[len(prefix):]))
			if err != nil {
				response.JSON(w, http.StatusUnauthorized, map[string]any{
					"error": response.ErrorBody{Code: response.CodeUnauthorized, Message: "token tidak valid"},
				})
				return
			}
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}

// WithActor menaruh Actor di context (dipakai JWT; dipisah untuk test).
func WithActor(ctx context.Context, a entity.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

type actorKey = typeKey

// WithActorFix HANYA TEST: middleware pemaca Actor tetap tanpa JWT.
func WithActorFix(a entity.Actor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), a)))
		})
	}
}

// rateLimiter fixed-window sederhana per kunci (security.md bagian 8:
// memori proses; pindah ke Redis saat scale-out).
type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	hits   map[string]*windowHits
}

type windowHits struct {
	start time.Time
	n     int
}

func NewRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{window: window, max: max, hits: map[string]*windowHits{}}
}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	w := rl.hits[key]
	if w == nil || now.Sub(w.start) >= rl.window {
		w = &windowHits{start: now}
		rl.hits[key] = w
	}
	w.n++
	return w.n <= rl.max
}

// RateLimit membatasi per user (atau IP bila belum login) sampai `max` per window.
// Lewat = 429 (security.md bagian 8).
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimitScoped(max, window, nil)
}

// RateLimitScoped seperti RateLimit, tapi hanya request yang lolos `keep`
// dihitung dan bisa kena 429 (mis. POST /runs 5/menit vs endpoint lain
// 60/menit, security.md bagian 8). keep nil = semua request dihitung.
// Kunci = user ID dari Actor; fallback IP untuk request tanpa actor.
func RateLimitScoped(max int, window time.Duration, keep func(*http.Request) bool) func(http.Handler) http.Handler {
	rl := NewRateLimiter(max, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if keep == nil || keep(r) {
				key := r.RemoteAddr
				if a, ok := ActorFrom(r.Context()); ok {
					key = string(a.UserID)
				}
				if !rl.Allow(key) {
					response.JSON(w, http.StatusTooManyRequests, map[string]any{
						"error": response.ErrorBody{Code: response.CodeRateLimited, Message: "terlalu banyak request"},
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
