// Package entity memuat struct dan aturan domain PaluKetok.
//
// Aturan dependency (plan/architecture.md bagian 2): package ini HANYA boleh
// import stdlib. Sumber bentuk struct: plan/erd.dbml.
package entity

import (
	"encoding/json"
	"time"
)

// ID adalah UUID sebagai string. Sengaja string supaya package ini tetap
// stdlib-only (tidak menarik google/uuid); konversi tipe dilakukan repository.
type ID string

// Actor adalah pelaku request, diambil dari JWT claim lalu diverifikasi ke
// profiles di server (plan/architecture.md bagian 3). Tidak pernah dari body.
type Actor struct {
	UserID  ID
	IsAdmin bool
	// Email dipakai untuk mengirim email tagihan ke gateway pembayaran
	// (Snap meneruskan email ini ke halaman bayar). Boleh kosong.
	Email string
}

// PlanTier mengikuti profiles.plan_tier (erd.dbml).
type PlanTier string

const (
	TierFree PlanTier = "free"
	TierPro  PlanTier = "pro"
)

// Profile adalah baris profiles (erd.dbml). credits + token_version hanya
// boleh diubah server (plan/security.md bagian 3).
type Profile struct {
	ID           ID
	Username     *string
	FullName     *string
	AvatarURL    *string
	PlanTier     PlanTier
	IsAdmin      bool
	Credits      int
	TokenVersion int
	CreatedAt    time.Time
}

// Stack adalah isi projects.stack (jsonb): {fe, be, db, payment}
// (erd.dbml baris 29, plan/security.md bagian 4).
type Stack struct {
	FE      string `json:"fe,omitempty"`
	BE      string `json:"be,omitempty"`
	DB      string `json:"db,omitempty"`
	Payment string `json:"payment,omitempty"`
}

// IsZero true bila keempat field kosong, sehingga repository bisa menulis NULL.
func (s Stack) IsZero() bool {
	return s == Stack{}
}

// ProjectStatus mengikuti CHECK projects.status (erd.dbml).
type ProjectStatus string

const (
	ProjectDraft    ProjectStatus = "draft"
	ProjectReady    ProjectStatus = "ready"
	ProjectArchived ProjectStatus = "archived"
)

// Project adalah baris projects (erd.dbml).
type Project struct {
	ID      ID
	OwnerID ID
	Name    string
	Brief   string
	Stack   Stack
	Status  ProjectStatus
	// Slug: bagian URL proyek yang enak dibaca (/projects/testing-jembut).
	// Unik per owner; beku sejak dibuat sehingga tautan yang dibagikan tidak
	// mati walau nama proyek diubah. Boleh kosong pada baris lama (migrasi
	// 0005 mengisinya; sisa kosong disajikan lewat UUID).
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Stage adalah enam tahap planner (plan/planner.md bagian 2). Dipakai untuk
// plan_runs.current_stage, artifacts.kind, dan decisions.stage.
type Stage string

const (
	StagePRD       Stage = "prd"
	StageERD       Stage = "erd"
	StageOpenAPI   Stage = "openapi"
	StageRoles     Stage = "roles"
	StageSecurity  Stage = "security"
	StageStructure Stage = "structure"
)

// Stages adalah urutan tetap pipeline. Tahap berikutnya selalu memakai hasil
// tahap sebelumnya (plan/planner.md bagian 2).
var Stages = []Stage{
	StagePRD, StageERD, StageOpenAPI, StageRoles, StageSecurity, StageStructure,
}

// TotalStages adalah nilai plan_runs.total (erd.dbml).
const TotalStages = 6

// RunStatus mengikuti CHECK plan_runs.status (erd.dbml).
type RunStatus string

const (
	RunQueued  RunStatus = "queued"
	RunRunning RunStatus = "running"
	RunDone    RunStatus = "done"
	RunFailed  RunStatus = "failed"
)

// PlanRun adalah baris plan_runs (erd.dbml).
type PlanRun struct {
	ID             ID
	ProjectID      ID
	UserID         ID
	Status         RunStatus
	CurrentStage   *Stage
	Progress       int
	Total          int
	Escalations    int
	CreditsCharged int
	Model          *string
	TokensIn       int
	TokensOut      int
	Cost           float64
	Error          *string
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
}

// ArtifactStatus mengikuti CHECK artifacts.status (erd.dbml).
type ArtifactStatus string

const (
	ArtifactDraft ArtifactStatus = "draft"
	ArtifactFinal ArtifactStatus = "final"
	ArtifactStale ArtifactStatus = "stale"
)

// Artifact adalah baris artifacts (erd.dbml). version + content selalu versi
// terbaru (plan/erd.md "Aturan versi artefak").
type Artifact struct {
	ID        ID
	RunID     ID
	UserID    ID
	Kind      Stage
	Content   string
	Version   int
	Status    ArtifactStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProjectMessage adalah baris project_messages: satu pesan percakapan
// perancangan milik satu proyek. Hanya 'user' dan 'assistant' yang sah —
// instruksi 'system' datang dari server (worker/prompt.go), bukan dari klien.
type ProjectMessage struct {
	ID        ID
	ProjectID ID
	UserID    ID
	Role      string
	Content   string
	// SuggestedStage usul tahap berikutnya dari model; kosong bila tidak ada.
	SuggestedStage string
	Model          string
	TokensIn       int
	TokensOut      int
	CreatedAt      time.Time
}

// ArtifactVersion adalah baris artifact_versions (erd.dbml). created_by_run_id
// hanya menunjuk run ASAL, tetap sama untuk semua versi hasil regenerate.
type ArtifactVersion struct {
	ID             ID
	ArtifactID     ID
	Version        int
	Content        string
	CreatedByRunID *ID
	CreatedAt      time.Time
}

// Gate mengikuti CHECK decisions.gate (erd.dbml) dan plan/planner.md bagian 4.
type Gate string

const (
	GateExecute        Gate = "EXECUTE"
	GateGatherEvidence Gate = "GATHER_EVIDENCE"
	GateEscalate       Gate = "ESCALATE"
)

// MaxGateRounds adalah batas putaran GATHER_EVIDENCE (plan/planner.md bagian 4).
const MaxGateRounds = 4

// Decision adalah baris decisions (erd.dbml). state dan criteria jsonb mentah:
// ke klien hanya dikirim ringkasan (plan/openapi.md bagian AI).
type Decision struct {
	ID            ID
	RunID         ID
	UserID        ID
	Stage         Stage
	Question      string
	State         json.RawMessage
	Criteria      json.RawMessage
	Gate          Gate
	Round         int
	Choice        *string
	Confidence    *float64
	Probabilities json.RawMessage
	JEVDecisionID *string
	CreatedAt     time.Time
}

// OrderStatus mengikuti CHECK orders.status (erd.dbml). `deny` dibuang saat
// pindah ke gateway pembayaran.
type OrderStatus string

const (
	OrderPending    OrderStatus = "pending"
	OrderSettlement OrderStatus = "settlement"
	OrderExpire     OrderStatus = "expire"
)

// Order adalah baris orders (erd.dbml). Sumber kedaluwarsa adalah expired_date
// dari respons gateway (Midtrans Snap), bukan timer sendiri
// (plan/auth_billing.md bagian 2.4).
type Order struct {
	ID             ID
	UserID         ID
	PackageID      string
	InvoiceNumber  string
	AmountIDR      int
	CreditsGranted int
	Status         OrderStatus
	RequestID      string
	PaymentURL     *string
	// GatewayToken adalah token halaman bayar dari gateway (Midtrans Snap
	// token), disimpan sebagai jejak audit. Bukan kunci transaksi: Midtrans
	// memakai invoice_number kita sebagai order_id.
	GatewayToken *string
	ExpiredDate  *time.Time
	PaidAt       *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// LedgerReason mengikuti CHECK credits_ledger.reason (erd.dbml).
type LedgerReason string

const (
	ReasonTopUp      LedgerReason = "topup"
	ReasonRun        LedgerReason = "run"
	ReasonRefund     LedgerReason = "refund"
	ReasonRegenerate LedgerReason = "regenerate"
	ReasonGrant      LedgerReason = "grant"
)

// LedgerRefType mengikuti CHECK credits_ledger.ref_type (erd.dbml).
type LedgerRefType string

const (
	RefOrder    LedgerRefType = "order"
	RefPlanRun  LedgerRefType = "plan_run"
	RefArtifact LedgerRefType = "artifact"
	RefLedger   LedgerRefType = "ledger"
	RefNone     LedgerRefType = "none"
)

// Ledger adalah baris credits_ledger (erd.dbml). Append-only: tidak ada UPDATE
// atau DELETE pada barisnya (plan/erd.md "Aturan kredit").
type Ledger struct {
	ID             ID
	UserID         ID
	Delta          int
	Reason         LedgerReason
	RefType        LedgerRefType
	RefID          *ID
	ActorID        *ID
	IdempotencyKey *string
	CreatedAt      time.Time
}

// NewPlanRun membuat run baru dengan nilai awal sesuai erd.dbml dan
// plan/auth_billing.md bagian 3.1 (status queued, total 6, credits_charged 1).
func NewPlanRun(id, projectID, userID ID, now time.Time) *PlanRun {
	return &PlanRun{
		ID:             id,
		ProjectID:      projectID,
		UserID:         userID,
		Status:         RunQueued,
		Progress:       0,
		Total:          TotalStages,
		CreditsCharged: 1,
		CreatedAt:      now,
	}
}
