// Package port memuat interface yang dipakai usecase. Implementasinya ada di
// internal/repository dan internal/adapter (plan/architecture.md bagian 2).
//
// Package ini hanya boleh import entity dan stdlib.
package port

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
)

// Clock dipakai usecase supaya waktu bisa deterministic saat dites
// (plan/architecture.md bagian 3).
type Clock interface {
	Now() time.Time
}

// Page membatasi hasil list. Cursor kosong berarti dari awal.
type Page struct {
	Limit  int
	Offset int
}

// ProfileRepo adalah akses baris profiles.
type ProfileRepo interface {
	GetByID(ctx context.Context, id entity.ID) (*entity.Profile, error)
	// SetPlanTier ubah plan_tier user; is_admin TIDAK bisa lewat API
	// (openapi.md PATCH /v1/admin/users/{id}).
	SetPlanTier(ctx context.Context, id entity.ID, tier entity.PlanTier) error
}

// ProjectRepo adalah akses projects, owner-scoped. Implementasi wajib memfilter
// owner_id (plan/architecture.md bagian 3).
type ProjectRepo interface {
	Create(ctx context.Context, p *entity.Project) error
	GetByID(ctx context.Context, id entity.ID) (*entity.Project, error)
	ListByOwner(ctx context.Context, ownerID entity.ID, page Page) ([]entity.Project, error)
	// ListAll semua project untuk admin (openapi.md GET /v1/admin/projects).
	ListAll(ctx context.Context, page Page) ([]entity.Project, error)
	Update(ctx context.Context, p *entity.Project) error
	// Archive mengubah status menjadi archived, bukan menghapus baris
	// (plan/erd.md "Aturan status", plan/openapi.md DELETE /v1/projects/{id}).
	Archive(ctx context.Context, id entity.ID, ownerID entity.ID, now time.Time) error
}

// RunRepo adalah akses plan_runs.
type RunRepo interface {
	GetByID(ctx context.Context, id entity.ID) (*entity.PlanRun, error)
	ListByProject(ctx context.Context, projectID entity.ID) ([]entity.PlanRun, error)
	// ListAllRuns untuk admin; filter kosong = tanpa filter
	// (openapi.md GET /v1/admin/runs?status=&user_id=).
	ListAllRuns(ctx context.Context, status string, userID entity.ID, page Page) ([]entity.PlanRun, error)
	// HasOtherRunning true bila project punya run running selain excludeID
	// (syarat regenerate planner.md bagian 7 dan penjaga worker Fase 6).
	HasOtherRunning(ctx context.Context, projectID, excludeID entity.ID) (bool, error)
	// HasRunning true bila project punya run berstatus running
	// (plan/erd.md: satu run running per project).
	HasRunning(ctx context.Context, projectID entity.ID) (bool, error)
	Update(ctx context.Context, r *entity.PlanRun) error
}

// QueuedRun adalah pasangan minimal yang dibutuhkan worker untuk mengklaim
// dan menjalankan satu run dari antrean restart.
type QueuedRun struct {
	ID        entity.ID
	ProjectID entity.ID
	UserID    entity.ID
}

// WorkerQueue adalah kumpulan operasi klaim penjadwalan run untuk worker
// (plan/workflow.md 22, 25; plan/security.md bagian 11). Semua operasi
// idempoten dan aman terhadap dua worker bersamaan.
type WorkerQueue interface {
	// ClaimQueued mengklaim SATU run 'queued' secara atomik:
	// UPDATE plan_runs SET status='running' WHERE id IN (
	//   SELECT id FROM plan_runs WHERE status='queued'
	//   ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)
	// Mengembalikan port.ErrNotFound bila antrean kosong. Dua worker tidak
	// pernah mendapat run yang sama.
	ClaimQueued(ctx context.Context) (*QueuedRun, error)
	// ListQueued mengembalikan run 'queued' yang belum ter-claim, urut
	// created_at (plan/security.md 11: start = run queued dimasukkan ulang).
	ListQueued(ctx context.Context) ([]QueuedRun, error)
	// RequeueRunning menyudahi restart: run 'running' sisa proses mati
	// ditandai failed dengan error "server_restart", kredit di-refund sekali
	// (idempoten lewat uq_ledger_one_refund_per_run).
	// Mengembalikan jumlah run yang ditandai.
	RequeueRunning(ctx context.Context, now time.Time) (int, error)
}

// ArtifactRepo adalah akses artifacts dan artifact_versions.
type ArtifactRepo interface {
	GetByID(ctx context.Context, id entity.ID) (*entity.Artifact, error)
	ListByRun(ctx context.Context, runID entity.ID) ([]entity.Artifact, error)
	Update(ctx context.Context, a *entity.Artifact) error
	// InsertVersion menyimpan versi lama sebelum artifacts ditimpa
	// (plan/erd.md "Aturan versi artefak").
	InsertVersion(ctx context.Context, v *entity.ArtifactVersion) error
	// HasVersionSet true bila (artifact_id, version) sudah ada di history.
	// Regenerate memakainya agar tidak menulis ulang versi lama hasil Upsert
	// pipeline (23505 uq_artifact_versions).
	HasVersionSet(ctx context.Context, artifactID entity.ID, version int) (bool, error)
	// MarkDownstreamStale set status='stale' semua artefak tahap sesudah
	// `after` pada run yang sama (planner.md bagian 7 regenerate).
	MarkDownstreamStale(ctx context.Context, runID entity.ID, after entity.Stage) error
}

// DecisionRepo mencatat jejak keputusan JEV (plan/planner.md bagian 4).
type DecisionRepo interface {
	Create(ctx context.Context, d *entity.Decision) error
	// ListByRun mengembalikan semua keputusan milik satu run, urut waktu
	// (openapi.md GET /v1/decisions?run_id=).
	ListByRun(ctx context.Context, runID entity.ID) ([]entity.Decision, error)
	// ListAll log JEV semua user untuk admin (openapi.md GET /v1/admin/decisions).
	ListAll(ctx context.Context, page Page) ([]entity.Decision, error)
	// NextRound mengembalikan nomor putaran berikutnya untuk (run_id, stage):
	// nomor tertinggi sebelumnya + 1, bukan mulai dari 1 lagi
	// (plan/erd.md "Aturan status", baris decisions).
	NextRound(ctx context.Context, runID entity.ID, stage entity.Stage) (int, error)
}

// OrderRepo adalah akses orders.
type OrderRepo interface {
	Create(ctx context.Context, o *entity.Order) error
	GetByID(ctx context.Context, id entity.ID) (*entity.Order, error)
	GetByInvoiceNumber(ctx context.Context, invoiceNumber string) (*entity.Order, error)
	GetByRequestID(ctx context.Context, requestID string) (*entity.Order, error)
	Update(ctx context.Context, o *entity.Order) error
}

// LedgerRepo memegang transaksi kredit. Transaksi dan SELECT ... FOR UPDATE ada
// di implementasi repository, bukan di usecase (plan/architecture.md bagian 3).
type LedgerRepo interface {
	// Charge mengambil kredit untuk sebuah run: potong saldo, insert ledger,
	// insert plan_runs, semuanya dalam satu transaksi
	// (plan/auth_billing.md bagian 3.1).
	ChargeForRun(ctx context.Context, userID entity.ID, run *entity.PlanRun) error
	// ChargeForRegenerate memotong 1 kredit dan mengembalikan id baris charge
	// (plan/auth_billing.md bagian 3.4, langkah 1-4).
	ChargeForRegenerate(ctx context.Context, userID entity.ID, artifactID entity.ID, charged int) (chargeID entity.ID, err error)
	// RefundRun me-refund sekali per run. Konflik unique index berarti refund
	// sudah ada dan bukan error (plan/auth_billing.md bagian 3.2).
	RefundRun(ctx context.Context, run *entity.PlanRun, credits int) error
	// RefundCharge me-refund sekali per baris charge regenerate
	// (plan/auth_billing.md bagian 3.4, refund).
	RefundCharge(ctx context.Context, userID entity.ID, chargeID entity.ID, credits int) error
	// Grant menambah kredit oleh admin. Idempoten lewat idempotency_key
	// (unique index uq_ledger_one_grant_per_key, erd.dbml). Implementasi wajib
	// memakai SELECT ... FOR UPDATE dan menolak saldo akhir negatif
	// (plan/security.md bagian 7).
	Grant(ctx context.Context, userID entity.ID, credits int, actorID entity.ID, idempotencyKey string) error
	// SettleTopUp men-settle order topup dalam satu transaksi: status
	// pending/expire -> settlement, insert ledger (+credits, reason topup,
	// ref_type order), tambah profiles.credits. Dipanggil lagi untuk order yang
	// sudah settlement = no-op (idempoten webhook, security.md 6 langkah 8-9).
	SettleTopUp(ctx context.Context, o *entity.Order) error
}

// RateLimiter membatasi request per user (plan/security.md bagian 8).
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
}
