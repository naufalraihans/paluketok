// Package entity memuat struct dan aturan domain PaluKetok.
//
// Aturan dependency (docs/plan/architecture.md bagian 2): package ini HANYA boleh
// import stdlib. Sumber bentuk struct: docs/plan/erd.dbml.
package entity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ID adalah UUID sebagai string. Sengaja string supaya package ini tetap
// stdlib-only (tidak menarik google/uuid); konversi tipe dilakukan repository.
type ID string

// Actor adalah pelaku request, diambil dari JWT claim lalu diverifikasi ke
// profiles di server (docs/plan/architecture.md bagian 3). Tidak pernah dari body.
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
// boleh diubah server (docs/plan/security.md bagian 3).
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
// (erd.dbml baris 29, docs/plan/security.md bagian 4).
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

// OutputFormat format keluaran tahap. Dibatasi tiga (keputusan pemilik): apa pun
// di luar ini ditolak worker, karena format lain tidak bisa diperiksa mesin.
type OutputFormat string

const (
	FormatMarkdown OutputFormat = "markdown"
	FormatJSON     OutputFormat = "json"
	FormatDBML     OutputFormat = "dbml"
)

// ValidOutputFormat true hanya untuk tiga format yang didukung.
func ValidOutputFormat(f OutputFormat) bool {
	switch f {
	case FormatMarkdown, FormatJSON, FormatDBML:
		return true
	}
	return false
}

// MinStages dan MaxStages batas jumlah tahap per run (docs/plan/planner.md):
// kurang dari dua bukan "tahap demi tahap", lebih dari dua belas bukan lagi
// dokumen perancangan yang bisa dibaca.
const (
	MinStages = 2
	MaxStages = 12
)

// ValidStageKey memeriksa kunci tahap cocok dengan plan_stages.key
// (^[a-z][a-z0-9_-]{1,30}$). Ditegakkan di sini supaya worker dan repository
// memakai aturan yang sama, bukan dua regexp yang bisa berbeda.
func ValidStageKey(k Stage) bool {
	s := string(k)
	if len(s) < 2 || len(s) > 31 {
		return false
	}
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

// PlanStage satu tahap planning milik satu run (tabel plan_stages, migrasi 0007).
//
// Sebelum ini tahap dipaku di enam konstanta. Sekarang AI judge menyusunnya per
// run, jadi jumlah dan namanya bebas dalam batas MinStages..MaxStages.
type PlanStage struct {
	ID           ID           `json:"id"`
	RunID        ID           `json:"run_id"`
	Key          Stage        `json:"key"`
	Label        string       `json:"label"`
	Position     int          `json:"position"`
	Intent       string       `json:"intent"`
	OutputFormat OutputFormat `json:"output_format"`
	CheckSpec    []byte       `json:"check_spec,omitempty"`
	DependsOn    []Stage      `json:"depends_on"`
	CreatedAt    time.Time    `json:"created_at"`
}

// DefaultPlanStages susunan awal enam tahap sebagai PlanStage.
//
// Dipakai saat: run lama tanpa baris plan_stages, judge tidak bisa dipakai, atau
// usulan judge ditolak. runID boleh kosong untuk pratinjau sebelum run ada.
func DefaultPlanStages(runID ID) []PlanStage {
	out := make([]PlanStage, 0, len(Stages))
	for i, s := range Stages {
		out = append(out, PlanStage{
			RunID:        runID,
			Key:          s,
			Label:        builtinStageLabel[s],
			Position:     i,
			Intent:       builtinStageIntent[s],
			OutputFormat: builtinStageFormat[s],
			CheckSpec:    builtinCheckSpec[s],
		})
	}
	for i := range out {
		out[i].DependsOn = builtinStageDeps[out[i].Key]
	}
	return out
}

var builtinStageLabel = map[Stage]string{
	StagePRD:       "PRD",
	StageERD:       "ERD",
	StageOpenAPI:   "OpenAPI",
	StageRoles:     "Role access",
	StageSecurity:  "Proteksi",
	StageStructure: "Struktur",
}

var builtinStageIntent = map[Stage]string{
	StagePRD:       "Rumuskan masalah, pengguna, fitur inti, dan batas proyek.",
	StageERD:       "Susun skema data: tabel, kolom, primary key, dan relasi.",
	StageOpenAPI:   "Tetapkan kontrak HTTP: path, method, dan respons tiap operasi.",
	StageRoles:     "Petakan peran dan hak akses untuk setiap path kontrak.",
	StageSecurity:  "Tetapkan trust boundary, validasi input, dan checklist proteksi.",
	StageStructure: "Tetapkan struktur folder dan aturan layer sesuai stack.",
}

var builtinStageFormat = map[Stage]OutputFormat{
	StagePRD:       FormatMarkdown,
	StageERD:       FormatDBML,
	StageOpenAPI:   FormatJSON,
	StageRoles:     FormatMarkdown,
	StageSecurity:  FormatMarkdown,
	StageStructure: FormatMarkdown,
}

var builtinStageDeps = map[Stage][]Stage{
	StagePRD:       {},
	StageERD:       {StagePRD},
	StageOpenAPI:   {StagePRD, StageERD},
	StageRoles:     {StagePRD, StageOpenAPI},
	StageSecurity:  {StageOpenAPI, StageERD, StageRoles},
	StageStructure: {StagePRD, StageOpenAPI, StageERD},
}

var builtinCheckSpec = map[Stage][]byte{
	StagePRD: []byte(`{"wajib_seksi":["Tujuan","Scope","Acceptance criteria","Di luar scope"]}`),
}

// NormalizePlanStages memeriksa skema yang datang dari luar (mis. usulan yang
// sudah disunting user) dan mengembalikannya dalam bentuk sah.
//
// Aturannya sama dengan yang ditegakkan worker; dipisah ke sini supaya HTTP
// handler bisa menolak lebih awal tanpa menunggu worker.
func NormalizePlanStages(in []PlanStage) ([]PlanStage, error) {
	if len(in) < MinStages || len(in) > MaxStages {
		return nil, fmt.Errorf("jumlah tahap %d di luar batas %d..%d", len(in), MinStages, MaxStages)
	}
	seen := make(map[Stage]bool, len(in))
	out := make([]PlanStage, 0, len(in))
	for i, s := range in {
		k := Stage(strings.TrimSpace(strings.ToLower(string(s.Key))))
		if !ValidStageKey(k) {
			return nil, fmt.Errorf("key tahap %q tidak sah", s.Key)
		}
		if seen[k] {
			return nil, fmt.Errorf("key tahap %q dipakai dua kali", k)
		}
		seen[k] = true
		if !ValidOutputFormat(s.OutputFormat) {
			return nil, fmt.Errorf("format %q tahap %s tidak dikenal", s.OutputFormat, k)
		}
		label := strings.TrimSpace(s.Label)
		if label == "" {
			label = string(k)
		}
		intent := strings.TrimSpace(s.Intent)
		if intent == "" {
			intent = "Susun artefak " + label + " untuk proyek ini."
		}
		out = append(out, PlanStage{
			RunID:        s.RunID,
			Key:          k,
			Label:        label,
			Position:     i,
			Intent:       intent,
			OutputFormat: s.OutputFormat,
			CheckSpec:    s.CheckSpec,
		})
	}
	// depends_on diperiksa setelah semua key terkumpul.
	for _, s := range in {
		for _, d := range s.DependsOn {
			dk := Stage(strings.TrimSpace(strings.ToLower(string(d))))
			if !ValidStageKey(dk) || !seen[dk] {
				return nil, fmt.Errorf("tahap %s bergantung pada %q yang tidak ada", s.Key, d)
			}
		}
	}
	for i, s := range in {
		var deps []Stage
		for _, d := range s.DependsOn {
			dk := Stage(strings.TrimSpace(strings.ToLower(string(d))))
			if !containsStage(deps, dk) {
				deps = append(deps, dk)
			}
		}
		out[i].DependsOn = deps
	}
	return out, nil
}

func containsStage(list []Stage, k Stage) bool {
	for _, s := range list {
		if s == k {
			return true
		}
	}
	return false
}

// Stage adalah kunci satu tahap planning. Sejak tahap disusun dinamis
// (plan_stages, migrasi 0007), nilainya TIDAK lagi terbatas pada enam konstanta
// di bawah: kunci bebas selama cocok pola plan_stages.key. Enam konstanta itu
// sekarang hanya SUSUNAN AWAL yang dipakai bila sebuah run belum punya skema
// sendiri, dan tetap menjadi kunci tahap bawaan yang validatornya sudah ada.
type Stage string

const (
	StagePRD       Stage = "prd"
	StageERD       Stage = "erd"
	StageOpenAPI   Stage = "openapi"
	StageRoles     Stage = "roles"
	StageSecurity  Stage = "security"
	StageStructure Stage = "structure"
)

// Stages adalah susunan awal enam tahap. Dipakai sebagai nilai awal saat run
// belum punya baris plan_stages (docs/plan/planner.md bagian 2); tahap
// berikutnya selalu memakai hasil tahap sebelumnya.
var Stages = []Stage{
	StagePRD, StageERD, StageOpenAPI, StageRoles, StageSecurity, StageStructure,
}

// TotalStages adalah jumlah tahap susunan awal (erd.dbml). Jumlah tahap run
// sebenarnya bisa berbeda: plan_runs.total diisi worker dari panjang skema.
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
// terbaru (docs/plan/erd.md "Aturan versi artefak").
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

// Gate mengikuti CHECK decisions.gate (erd.dbml) dan docs/plan/planner.md bagian 4.
type Gate string

const (
	GateExecute        Gate = "EXECUTE"
	GateGatherEvidence Gate = "GATHER_EVIDENCE"
	GateEscalate       Gate = "ESCALATE"
)

// MaxGateRounds adalah batas putaran GATHER_EVIDENCE (docs/plan/planner.md bagian 4).
const MaxGateRounds = 4

// Decision adalah baris decisions (erd.dbml). state dan criteria jsonb mentah:
// ke klien hanya dikirim ringkasan (docs/plan/openapi.md bagian AI).
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
// (docs/plan/auth_billing.md bagian 2.4).
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
// atau DELETE pada barisnya (docs/plan/erd.md "Aturan kredit").
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
// docs/plan/auth_billing.md bagian 3.1 (status queued, total 6, credits_charged 1).
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
