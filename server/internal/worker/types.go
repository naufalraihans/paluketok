// Kontrak dependensi worker (pipeline.go memuat narasi plan).
package worker

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// RunStore kebutuhan worker atas plan_runs.
type RunStore interface {
	// GetFull mengembalikan PlanRun penuh (credits_charged untuk refund).
	GetFull(ctx context.Context, id entity.ID) (*entity.PlanRun, error)
	ClaimQueued(ctx context.Context) (*port.QueuedRun, error)
	// SaveProgress menulis current_stage (di AWAL tahap) + progress + status
	// dalam satu UPDATE (planner.md bagian 1 langkah a0/f).
	SaveProgress(ctx context.Context, id entity.ID, stage entity.Stage, progress int, status entity.RunStatus) error
	// SaveTotal menulis plan_runs.total = jumlah tahap sebenarnya milik run.
	// Tahap disusun dinamis, jadi total tidak lagi selalu 6.
	SaveTotal(ctx context.Context, id entity.ID, total int) error
	// Finish menutup run: done/failed + error + finished_at.
	Finish(ctx context.Context, id entity.ID, status entity.RunStatus, runErr string, now time.Time) error
	ListQueued(ctx context.Context) ([]port.QueuedRun, error)
	RequeueRunning(ctx context.Context, now time.Time) (int, error)
	// HasOtherRunning penjaga satu-running per project (planner.md bagian 8).
	// excludeID = run yang sedang diproses, agar tidak menghitung dirinya.
	HasOtherRunning(ctx context.Context, projectID, excludeID entity.ID) (bool, error)
}

// ArtifactStore kebutuhan worker atas artifacts + artifact_versions.
// Upsert menulis artifacts dan versi (uq_artifacts_run_kind).
type ArtifactStore interface {
	Upsert(ctx context.Context, a *entity.Artifact) error
	// ListByRun dipakai regenerate (planner.md bagian 7) untuk konteks prompt.
	ListByRun(ctx context.Context, runID entity.ID) ([]entity.Artifact, error)
}

// ProjectsSource ambil brief/stack untuk prompt; dibutuhkan worker.
type ProjectsSource interface {
	GetByID(ctx context.Context, id entity.ID) (*entity.Project, error)
}

// LedgerStore: refund run idempoten di repo (uq_ledger_one_refund_per_run).
// Jumlah kredit diambil dari run.CreditsCharged (sumber tunggal).
type LedgerStore interface {
	RefundRun(ctx context.Context, run *entity.PlanRun, credits int) error
}

// DecisionsStore mencatat setiap panggilan gate JEV (planner.md bagian 4).
type DecisionsStore interface {
	Create(ctx context.Context, d *entity.Decision) error
}

// PlanStageStore kebutuhan worker atas plan_stages: skema tahap milik run.
// Daftar kosong berarti run lama tanpa skema, dan worker memakai susunan awal.
type PlanStageStore interface {
	ListByRun(ctx context.Context, runID entity.ID) ([]entity.PlanStage, error)
	// ReplaceByRun menulis skema satu run dalam satu transaksi (hapus lalu isi).
	ReplaceByRun(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error
}

// LLM dan Judge = port asli, tanpa duplikasi.
type LLM = port.LLM
type Judge = port.Judge
