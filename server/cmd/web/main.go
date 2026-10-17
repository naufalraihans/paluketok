// Command web: composition root PaluKetok server (plan/workflow.md 20).
// config -> pool -> repos+adapter -> usecases -> router -> http.Server.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"paluketok/server/internal/adapter/judge"
	"paluketok/server/internal/adapter/llm"
	"paluketok/server/internal/adapter/payment/midtrans"
	"paluketok/server/internal/delivery/httpapi"
	"paluketok/server/internal/delivery/middleware"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/repository"
	"paluketok/server/internal/usecase"
	"paluketok/server/internal/worker"
	"paluketok/server/pkg/config"
	"paluketok/server/pkg/database"
	"paluketok/server/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

// run bertahan hidup sampai sinyal; err hanya dari kegagalan start/serve.
func run() error {
	srv, err := newServer(".env")
	if err != nil {
		return err
	}
	log := logger.Setup(srv.env)

	errCh := make(chan error, 1)
	go func() {
		log.Info("server mulai", "addr", srv.http.Addr, "env", srv.env)
		if err := srv.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutdown: menunggu koneksi selesai")
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.http.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("server berhenti")
	return nil
}

// server siap dijalankan: http.Server + env (untuk logger pemanggil).
type server struct {
	http *http.Server
	env  string
}

// newServer merakit semuanya. Dipisah dari run() agar composition root
// bisa dites sungguhan (cmd/web/main_test.go menunduh /healthz dari sini).
func newServer(envFile string) (*server, error) {
	cfg, err := config.Load(envFile)
	if err != nil {
		return nil, err
	}

	pool, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	// Repozitori dan adapter (Fase 4).
	repos := repository.New(pool)
	payment := midtrans.New(cfg.MidtransAuthBaseURL, cfg.MidtransAPIBaseURL, cfg.MidtransServerKey)
	judgeClient := judge.New(cfg.JEVBaseURL, cfg.JEVAPIKey) // fallback env; slot "judge" menang bila terisi
	llmClient := llm.New()                                  // provider per slot dari DB (planner.md 9)

	// ActorisAdmin: dibaca dari profiles oleh handler via repo. Sederhana F5:
	// isAdmin dibiarkan false; endpoint admin F7 menyusul. (deviasi dicatat.)
	apps := &usecase.Deps{
		Profiles: repos.Profiles, Projects: repos.Projects, Runs: repos.Runs,
		Artifacts: repos.Artifacts, Decisions: repos.Decisions, Orders: repos.Orders,
		Ledger:  repos.Ledger,
		Payment: payment, Signatures: payment, Verifier: payment, Notifications: payment,
		// Chat konteks proyek memakai slot `planner` yang sama dengan worker.
		// resolvedLLM membaca ulang ai_settings per panggilan, jadi ganti key
		// dari admin berlaku tanpa restart.
		Messages: repos.Messages,
		LLM:      resolvedLLM{client: llmClient, store: repos.AISettings},
		// Semua field wajib Deps terisi (panic NewProjectUsecase kalau tidak):
		Clock: clockReal{}, IDs: idsReal{},
	}

	runUC := usecase.NewRunUsecase(*apps)
	projUC := usecase.NewProjectUsecase(*apps)
	// Chat resolusi proyek lewat jalur yang sama dengan endpoint proyek:
	// satu aturan 403-vs-404 dan satu penerjemah slug.
	chatUC := usecase.NewChatUsecase(*apps)
	chatUC.SetResolver(projUC)
	h := &httpapi.Handlers{
		Projects: projUC,
		Runs:     runUC,
		Credits:  usecase.NewCreditUsecase(*apps),
		Orders:   usecase.NewOrderUsecase(*apps),
		Admin:    usecase.NewAdminUsecase(*apps, repos.AISettings),
		Chat:     chatUC,
	}
	verifier := middleware.NewVerifier(cfg.SupabaseURL, cfg.JWKSURL)
	verifier.IsAdmin = repos.Profiles.IsAdmin // profiles.is_admin (F7)
	router := httpapi.NewRouter(httpapi.RouterOptions{
		Handlers: h,
		Verifier: verifier,
	})

	// Worker planner (Fase 6): pool 4, antrean 20 (security.md 8/11). Start
	// tanpa worker tetap aman: klaim DB mengosongkan antrean.
	runAdapter := workerRunAdapter{repos: repos}
	pipe := worker.New(resolvedLLM{client: llmClient, store: repos.AISettings}, resolvedJudge{client: judgeClient, store: repos.AISettings},
		runAdapter, repos.Projects, repos.Artifacts, repos.Ledger, repos.Decisions, idsReal{},
		func() time.Time { return time.Now().UTC() }, logger.Setup(cfg.AppEnv))
	wp, err := worker.NewPool(pipe, runAdapter, idsReal{}, func() time.Time { return time.Now().UTC() }, logger.Setup(cfg.AppEnv))
	if err != nil {
		return nil, fmt.Errorf("worker: %w", err)
	}
	h.Submitter = wp
	runUC.SetRegenerator(pipe) // POST /v1/artifacts/{id}/regenerate (planner.md 7)
	go wp.Start(context.Background(), wp.Warmup(context.Background()))

	return &server{
		http: &http.Server{
			Addr:              ":" + cfg.AppPort,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second, // security.md: lambatloris
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second, // LLM 180s lewat worker, bukan request sinkron
			IdleTimeout:       60 * time.Second,
		},
		env: cfg.AppEnv,
	}, nil
}

type clockReal struct{}

func (clockReal) Now() time.Time { return time.Now().UTC() }

// idsReal membuat UUID v4 dari crypto/rand (pola internal/repository/uuid.go).
// Usecase tidak boleh impor uuid, jadi implementasi hidup di composition root.
type idsReal struct{}

func (idsReal) New() entity.ID {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return entity.ID(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}
