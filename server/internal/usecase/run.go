package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// RunUsecase membuat dan membaca plan_runs.
type RunUsecase struct {
	deps        Deps
	regenerator Regenerator
	logger      *slog.Logger
}

func NewRunUsecase(d Deps) *RunUsecase { return &RunUsecase{deps: d, logger: slog.Default()} }

// clock dan ids aksesDependencies yang sama dipakai regenerate.
func (u *RunUsecase) clock() time.Time { return u.deps.Clock.Now() }
func (u *RunUsecase) ids() IDGenerator { return u.deps.IDs }

// CreateRun memulai run baru untuk project: cek owner, cek satu-running,
// lalu potong 1 kredit dalam satu transaksi di repository
// (plan/auth_billing.md 3.1 urutan FOR UPDATE -> cek -> insert run -> ledger).
func (u *RunUsecase) CreateRun(ctx context.Context, actor entity.Actor, projectID entity.ID) (*entity.PlanRun, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	if !entity.ValidID(string(projectID)) {
		return nil, ErrNotFound
	}
	p, err := u.deps.Projects.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	if p.OwnerID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}

	running, err := u.deps.Runs.HasRunning(ctx, projectID)
	if err != nil {
		return nil, wrap(err)
	}
	if running {
		// Instruksi 13c: tolak bila ada run running di project sama.
		return nil, ErrValidation
	}

	run := entity.NewPlanRun(u.deps.IDs.New(), projectID, actor.UserID, u.deps.Clock.Now())
	if err := u.deps.Ledger.ChargeForRun(ctx, actor.UserID, run); err != nil {
		return nil, wrap(err)
	}
	return run, nil
}

// GetRun membaca status run milik actor (openapi.md GET /v1/runs/{id}).
func (u *RunUsecase) GetRun(ctx context.Context, actor entity.Actor, runID entity.ID) (*entity.PlanRun, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	if !entity.ValidID(string(runID)) {
		return nil, ErrNotFound
	}
	run, err := u.deps.Runs.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	if run.UserID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return run, nil
}

// ListRunArtifacts membaca artefak milik run (openapi.md GET /v1/runs/{id}/artifacts).
func (u *RunUsecase) ListRunArtifacts(ctx context.Context, actor entity.Actor, runID entity.ID) ([]entity.Artifact, error) {
	if _, err := u.GetRun(ctx, actor, runID); err != nil {
		return nil, err
	}
	list, err := u.deps.Artifacts.ListByRun(ctx, runID)
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// ListDecisions mengembalikan jejak keputusan JEV milik actor untuk satu run
// (openapi.md GET /v1/decisions?run_id=). State dan criteria mentah TIDAK
// ikut (openapi.md): pemangkasnya di handler.
func (u *RunUsecase) ListDecisions(ctx context.Context, actor entity.Actor, runID entity.ID) ([]entity.Decision, error) {
	if _, err := u.GetRun(ctx, actor, runID); err != nil {
		return nil, err
	}
	list, err := u.deps.Decisions.ListByRun(ctx, runID)
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// LatestRun mengembalikan run TERBARU milik project (nil bila belum ada run).
//
// Frontend butuh ini supaya status generate tidak lagi bergantung pada run id
// yang cuma tersimpan di sessionStorage: proyek dibuka di browser/perangkat
// lain tetap menampilkan progres yang benar.
//
// ponytail: satu run terbaru, bukan riwayat penuh. Tambah endpoint list bila
// halaman riwayat run benar-benar dibangun.
func (u *RunUsecase) LatestRun(ctx context.Context, actor entity.Actor, projectID entity.ID) (*entity.PlanRun, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	p, err := u.deps.Projects.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	if p.OwnerID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	runs, err := u.deps.Runs.ListByProject(ctx, projectID)
	if err != nil {
		return nil, wrap(err)
	}
	if len(runs) == 0 {
		return nil, nil
	}
	newest := runs[0]
	for _, r := range runs[1:] {
		if r.CreatedAt.After(newest.CreatedAt) {
			newest = r
		}
	}
	return &newest, nil
}

// FailRun menandai run gagal dan me-refund kreditnya sekali
// (auth_billing.md 3.2). Dipakai worker, bukan endpoint publik.
func (u *RunUsecase) FailRun(ctx context.Context, runID entity.ID, reason string) error {
	if !entity.ValidID(string(runID)) {
		return ErrNotFound
	}
	run, err := u.deps.Runs.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return wrap(err)
	}
	if run.Status == entity.RunFailed {
		return nil // sudah gagal; refund idempoten di sisi ledger
	}
	if run.Status == entity.RunDone {
		return fmt.Errorf("%w: run done tidak boleh difail", ErrConflict)
	}
	run.Status = entity.RunFailed
	run.Error = &reason
	if err := u.deps.Runs.Update(ctx, run); err != nil {
		return wrap(err)
	}
	if run.CreditsCharged > 0 {
		if err := u.deps.Ledger.RefundRun(ctx, run, run.CreditsCharged); err != nil {
			return wrap(err)
		}
	}
	return nil
}
