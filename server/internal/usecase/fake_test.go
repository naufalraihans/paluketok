package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Fake-clock dan fake-id deterministik untuk tes.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

type seqIDs struct{ n int }

func (g *seqIDs) New() entity.ID {
	g.n++
	// UUID v4-shaped, cukup untuk lolos entity.ValidID tanpa dependensi uuid.
	return entity.ID(fmt.Sprintf("00000000-0000-4000-8000-%012d", g.n))
}

// fakeProfileRepo menyimpan profiles di memori.
type fakeProfileRepo struct {
	byID map[entity.ID]*entity.Profile
}

func newFakeProfiles(list ...*entity.Profile) *fakeProfileRepo {
	r := &fakeProfileRepo{byID: map[entity.ID]*entity.Profile{}}
	for _, p := range list {
		r.byID[p.ID] = p
	}
	return r
}

func (r *fakeProfileRepo) GetByID(_ context.Context, id entity.ID) (*entity.Profile, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

// fakeProjectRepo menyimpan projects di memori dan memfilter owner seperti
// repository sungguhan.
type fakeProjectRepo struct {
	byID  map[entity.ID]*entity.Project
	calls []string
}

func newFakeProjects() *fakeProjectRepo {
	return &fakeProjectRepo{byID: map[entity.ID]*entity.Project{}}
}

func (r *fakeProjectRepo) Create(_ context.Context, p *entity.Project) error {
	if _, dup := r.byID[p.ID]; dup {
		return port.ErrConflict
	}
	cp := *p
	r.byID[p.ID] = &cp
	r.calls = append(r.calls, "create")
	return nil
}

func (r *fakeProjectRepo) GetByID(_ context.Context, id entity.ID) (*entity.Project, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakeProjectRepo) ListByOwner(_ context.Context, ownerID entity.ID, page port.Page) ([]entity.Project, error) {
	var out []entity.Project
	for _, p := range r.byID {
		if p.OwnerID == ownerID {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if page.Offset >= len(out) {
		return nil, nil
	}
	out = out[page.Offset:]
	if page.Limit < len(out) {
		out = out[:page.Limit]
	}
	return out, nil
}

func (r *fakeProjectRepo) Update(_ context.Context, p *entity.Project) error {
	if _, ok := r.byID[p.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *p
	r.byID[p.ID] = &cp
	r.calls = append(r.calls, "update")
	return nil
}

func (r *fakeProjectRepo) Archive(_ context.Context, id entity.ID, ownerID entity.ID, now time.Time) error {
	p, ok := r.byID[id]
	if !ok {
		return port.ErrNotFound
	}
	if p.OwnerID != ownerID {
		return port.ErrForbidden
	}
	p.Status = entity.ProjectArchived
	p.UpdatedAt = now
	r.calls = append(r.calls, "archive")
	return nil
}

// fakeRunRepo menyimpan plan_runs di memori.
type fakeRunRepo struct {
	byID map[entity.ID]*entity.PlanRun
}

func newFakeRuns() *fakeRunRepo { return &fakeRunRepo{byID: map[entity.ID]*entity.PlanRun{}} }

func (r *fakeRunRepo) GetByID(_ context.Context, id entity.ID) (*entity.PlanRun, error) {
	run, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *run
	return &cp, nil
}

func (r *fakeRunRepo) ListByProject(_ context.Context, projectID entity.ID) ([]entity.PlanRun, error) {
	var out []entity.PlanRun
	for _, run := range r.byID {
		if run.ProjectID == projectID {
			out = append(out, *run)
		}
	}
	return out, nil
}

func (r *fakeRunRepo) HasRunning(_ context.Context, projectID entity.ID) (bool, error) {
	for _, run := range r.byID {
		if run.ProjectID == projectID && run.Status == entity.RunRunning {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRunRepo) Update(_ context.Context, run *entity.PlanRun) error {
	if _, ok := r.byID[run.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *run
	r.byID[run.ID] = &cp
	return nil
}

// fakeLedgerRepo meniru transaksi kredit: cek saldo, potong, catat baris.
// Jaring unique index ditiru lewat refunded/charged set + grants map.
type fakeLedgerRepo struct {
	profiles *fakeProfileRepo
	ledger   []entity.Ledger
	runs     *fakeRunRepo
	orders   *fakeOrderRepo
	refunded map[entity.ID]bool
	grants   map[string]bool
	failNext error
	clock    *fakeClock
}

func newFakeLedger(profiles *fakeProfileRepo, runs *fakeRunRepo, orders *fakeOrderRepo, clock *fakeClock) *fakeLedgerRepo {
	return &fakeLedgerRepo{profiles: profiles, runs: runs, orders: orders, refunded: map[entity.ID]bool{}, clock: clock}
}

// Grant menambah kredit. Idempoten lewat kunci idempotency_key, meniru
// unique index uq_ledger_one_grant_per_key (erd.dbml): replay = no-op.
func (l *fakeLedgerRepo) Grant(_ context.Context, userID entity.ID, credits int, actorID entity.ID, idempotencyKey string) error {
	if l.grants == nil {
		l.grants = map[string]bool{}
	}
	if l.grants[idempotencyKey] {
		return nil
	}
	p, ok := l.profiles.byID[userID]
	if !ok {
		return port.ErrNotFound
	}
	l.grants[idempotencyKey] = true
	p.Credits += credits
	aid, key := actorID, idempotencyKey
	l.ledger = append(l.ledger, entity.Ledger{
		UserID: userID, Delta: credits, Reason: entity.ReasonGrant,
		RefType: entity.RefNone, ActorID: &aid, IdempotencyKey: &key,
	})
	return nil
}

// SettleTopUp men-settle order sekali: pending/expire -> settlement,
// tambah kredit, catat ledger reason topup ref_type order
// (meniru uq_ledger_one_topup_per_order + transisi satu arah auth_billing 2.3).
func (l *fakeLedgerRepo) SettleTopUp(_ context.Context, o *entity.Order) error {
	if o.Status == entity.OrderSettlement {
		return nil
	}
	if l.failNext != nil {
		return l.failNext
	}
	p, ok := l.profiles.byID[o.UserID]
	if !ok {
		return port.ErrNotFound
	}
	now := l.clock.Now()
	o.Status = entity.OrderSettlement
	o.PaidAt = &now
	o.UpdatedAt = now
	p.Credits += o.CreditsGranted
	l.ledger = append(l.ledger, entity.Ledger{
		UserID: o.UserID, Delta: o.CreditsGranted, Reason: entity.ReasonTopUp,
		RefType: entity.RefOrder, RefID: &o.ID,
	})
	// tulis kembali ke penyimpanan order seperti UPDATE transaksional repo nyata
	return l.orders.Update(context.Background(), o)
}

func (l *fakeLedgerRepo) ChargeForRun(_ context.Context, userID entity.ID, run *entity.PlanRun) error {
	if l.failNext != nil {
		return l.failNext
	}
	p, ok := l.profiles.byID[userID]
	if !ok {
		return port.ErrNotFound
	}
	if p.Credits < run.CreditsCharged {
		return port.ErrInsufficientCredits
	}
	p.Credits -= run.CreditsCharged
	l.runs.byID[run.ID] = run
	l.ledger = append(l.ledger, entity.Ledger{
		UserID: userID, Delta: -run.CreditsCharged, Reason: entity.ReasonRun,
		RefType: entity.RefPlanRun, RefID: &run.ID,
	})
	return nil
}

func (l *fakeLedgerRepo) ChargeForRegenerate(_ context.Context, userID entity.ID, artifactID entity.ID, charged int) (entity.ID, error) {
	p, ok := l.profiles.byID[userID]
	if !ok {
		return "", port.ErrNotFound
	}
	if p.Credits < charged {
		return "", port.ErrInsufficientCredits
	}
	p.Credits -= charged
	id := entity.ID(fmt.Sprintf("00000000-0000-4000-9000-%012d", len(l.ledger)+1))
	l.ledger = append(l.ledger, entity.Ledger{
		ID: id, UserID: userID, Delta: -charged, Reason: entity.ReasonRegenerate,
		RefType: entity.RefArtifact, RefID: &artifactID,
	})
	return id, nil
}

func (l *fakeLedgerRepo) RefundRun(_ context.Context, run *entity.PlanRun, credits int) error {
	if l.refunded[run.ID] {
		return nil // konflik unique index = refund sudah ada, bukan error
	}
	l.refunded[run.ID] = true
	p, ok := l.profiles.byID[run.UserID]
	if !ok {
		return port.ErrNotFound
	}
	p.Credits += credits
	l.ledger = append(l.ledger, entity.Ledger{
		UserID: run.UserID, Delta: credits, Reason: entity.ReasonRefund,
		RefType: entity.RefPlanRun, RefID: &run.ID,
	})
	return nil
}

func (l *fakeLedgerRepo) RefundCharge(_ context.Context, userID entity.ID, chargeID entity.ID, credits int) error {
	if l.refunded[chargeID] {
		return nil
	}
	l.refunded[chargeID] = true
	p, ok := l.profiles.byID[userID]
	if !ok {
		return port.ErrNotFound
	}
	p.Credits += credits
	l.ledger = append(l.ledger, entity.Ledger{
		UserID: userID, Delta: credits, Reason: entity.ReasonRefund,
		RefType: entity.RefLedger, RefID: &chargeID,
	})
	return nil
}

// fakeArtifactRepo menyimpan artifacts dan versions.
type fakeArtifactRepo struct {
	byID     map[entity.ID]*entity.Artifact
	versions []entity.ArtifactVersion
}

func newFakeArtifacts() *fakeArtifactRepo {
	return &fakeArtifactRepo{byID: map[entity.ID]*entity.Artifact{}}
}

func (r *fakeArtifactRepo) GetByID(_ context.Context, id entity.ID) (*entity.Artifact, error) {
	a, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *fakeArtifactRepo) ListByRun(_ context.Context, runID entity.ID) ([]entity.Artifact, error) {
	var out []entity.Artifact
	for _, a := range r.byID {
		if a.RunID == runID {
			out = append(out, *a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out, nil
}

func (r *fakeArtifactRepo) Update(_ context.Context, a *entity.Artifact) error {
	if _, ok := r.byID[a.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *a
	r.byID[a.ID] = &cp
	return nil
}

// InsertVersion menegakkan unique (artifact_id, version) seperti index
// uq_artifact_versions di Postgres (erd.dbml): duplikat = port.ErrConflict,
// bukan diam-diam. Meniru 23505 agar bug tumbukan versi regenerate terbaca.
func (r *fakeArtifactRepo) InsertVersion(_ context.Context, v *entity.ArtifactVersion) error {
	for _, ex := range r.versions {
		if ex.ArtifactID == v.ArtifactID && ex.Version == v.Version {
			return port.ErrConflict
		}
	}
	r.versions = append(r.versions, *v)
	return nil
}

// HasVersionSet true bila pasangan (artifact_id, version) sudah ada di history.
func (r *fakeArtifactRepo) HasVersionSet(_ context.Context, artifactID entity.ID, version int) (bool, error) {
	for _, ex := range r.versions {
		if ex.ArtifactID == artifactID && ex.Version == version {
			return true, nil
		}
	}
	return false, nil
}

// HasVersionSet versi fakeRegenRepo (regenerate_test.go), view sama.
func (f *fakeRegenRepo) HasVersionSet(_ context.Context, artifactID entity.ID, version int) (bool, error) {
	for _, ex := range f.versions {
		if ex.ArtifactID == artifactID && ex.Version == version {
			return true, nil
		}
	}
	return false, nil
}

// fakeDecisionRepo menyimpan decisions dan menghitung putaran.
type fakeDecisionRepo struct {
	rows []entity.Decision
}

func (r *fakeDecisionRepo) Create(_ context.Context, d *entity.Decision) error {
	r.rows = append(r.rows, *d)
	return nil
}

func (r *fakeDecisionRepo) ListByRun(_ context.Context, runID entity.ID) ([]entity.Decision, error) {
	var out []entity.Decision
	for _, d := range r.rows {
		if d.RunID == runID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (r *fakeDecisionRepo) NextRound(_ context.Context, runID entity.ID, stage entity.Stage) (int, error) {
	max := 0
	for _, d := range r.rows {
		if d.RunID == runID && d.Stage == stage && d.Round > max {
			max = d.Round
		}
	}
	return max + 1, nil
}

// fakeOrderRepo menyimpan orders dan menegakkan unique invoice_number/request_id.
type fakeOrderRepo struct {
	byID map[entity.ID]*entity.Order
}

func newFakeOrders() *fakeOrderRepo { return &fakeOrderRepo{byID: map[entity.ID]*entity.Order{}} }

func (r *fakeOrderRepo) Create(_ context.Context, o *entity.Order) error {
	for _, ex := range r.byID {
		if ex.InvoiceNumber == o.InvoiceNumber || ex.RequestID == o.RequestID {
			return port.ErrConflict
		}
	}
	cp := *o
	r.byID[o.ID] = &cp
	return nil
}

func (r *fakeOrderRepo) GetByID(_ context.Context, id entity.ID) (*entity.Order, error) {
	o, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *o
	return &cp, nil
}

func (r *fakeOrderRepo) GetByInvoiceNumber(_ context.Context, inv string) (*entity.Order, error) {
	for _, o := range r.byID {
		if o.InvoiceNumber == inv {
			cp := *o
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (r *fakeOrderRepo) GetByRequestID(_ context.Context, reqID string) (*entity.Order, error) {
	for _, o := range r.byID {
		if o.RequestID == reqID {
			cp := *o
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (r *fakeOrderRepo) Update(_ context.Context, o *entity.Order) error {
	if _, ok := r.byID[o.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *o
	r.byID[o.ID] = &cp
	return nil
}

// fakePayment, fakeSignature, fakeParser untuk DOKU (13b-13d).
type fakePayment struct {
	resp     *port.PaymentResponse
	err      error
	lastReq  port.PaymentRequest
	statuses map[string]*port.PaymentStatus
}

func (f *fakePayment) CreateOrder(_ context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	f.lastReq = req
	return f.resp, f.err
}

func (f *fakePayment) CheckStatus(_ context.Context, inv string) (*port.PaymentStatus, error) {
	if f.statuses == nil {
		return nil, port.ErrUnavailable
	}
	s, ok := f.statuses[inv]
	if !ok {
		return nil, port.ErrNotFound
	}
	return s, nil
}

type fakeSignature struct {
	err error
	ops []string
}

func (f *fakeSignature) Verify(_ port.WebhookInput) error {
	f.ops = append(f.ops, "verify")
	return f.err
}

type fakeParser struct {
	out *port.Notification
	err error
}

func (f *fakeParser) Parse(_ []byte) (*port.Notification, error) { return f.out, f.err }

// fakes mengumpulkan semua fake supaya tes bisa memeriksa efek samping.
type fakes struct {
	profiles  *fakeProfileRepo
	projects  *fakeProjectRepo
	runs      *fakeRunRepo
	ledger    *fakeLedgerRepo
	orders    *fakeOrderRepo
	artifacts *fakeArtifactRepo
	decisions *fakeDecisionRepo
	payment   *fakePayment
	signature *fakeSignature
	parser    *fakeParser
}

// newTestCore merakit Deps lengkap dengan fake deterministik.
func newTestCore() (Deps, *fakeClock, *fakes) {
	now := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	clock := &fakeClock{t: now}
	profiles := newFakeProfiles()
	runs := newFakeRuns()
	projects := newFakeProjects()
	orders := newFakeOrders()
	ledger := newFakeLedger(profiles, runs, orders, clock)
	f := &fakes{
		profiles:  profiles,
		projects:  projects,
		runs:      runs,
		ledger:    ledger,
		orders:    orders,
		artifacts: newFakeArtifacts(),
		decisions: &fakeDecisionRepo{},
		payment:   &fakePayment{resp: &port.PaymentResponse{PaymentURL: "https://pay.doku/x", ExpiredDate: "20261014100000"}},
		signature: &fakeSignature{},
		parser:    &fakeParser{},
	}
	deps := Deps{
		Profiles:      profiles,
		Projects:      projects,
		Runs:          runs,
		Artifacts:     f.artifacts,
		Decisions:     f.decisions,
		Orders:        f.orders,
		Ledger:        ledger,
		Payment:       f.payment,
		Signature:     f.signature,
		Notifications: f.parser,
		Clock:         clock,
		IDs:           &seqIDs{},
	}
	return deps, clock, f
}

// newTestDeps dipertahankan untuk tes lama (project_test.go).
func newTestDeps() (Deps, *fakeClock, *fakeProfileRepo, *fakeProjectRepo, *fakeRunRepo, *fakeLedgerRepo) {
	deps, clock, f := newTestCore()
	return deps, clock, f.profiles, f.projects, f.runs, f.ledger
}

// seedProfile menambah profile dengan saldo kredit tertentu.
func seedProfile(r *fakeProfileRepo, id entity.ID, credits int) *entity.Profile {
	p := &entity.Profile{ID: id, PlanTier: entity.TierFree, Credits: credits}
	r.byID[id] = p
	return p
}

const (
	userA = entity.ID("11111111-1111-4111-8111-111111111111")
	userB = entity.ID("22222222-2222-4222-8222-222222222222")
	admin = entity.ID("33333333-3333-4333-8333-333333333333")
)

func (f *fakeArtifactRepo) MarkDownstreamStale(_ context.Context, _ entity.ID, _ entity.Stage) error {
	return nil
}

func (f *fakeDecisionRepo) ListAll(_ context.Context, _ port.Page) ([]entity.Decision, error) {
	return nil, nil
}

func (f *fakeProfileRepo) SetPlanTier(_ context.Context, _ entity.ID, _ entity.PlanTier) error {
	return nil
}

func (f *fakeProjectRepo) ListAll(_ context.Context, _ port.Page) ([]entity.Project, error) {
	return nil, nil
}

func (f *fakeRunRepo) ListAllRuns(_ context.Context, _ string, _ entity.ID, _ port.Page) ([]entity.PlanRun, error) {
	return nil, nil
}

func (f *fakeRunRepo) HasOtherRunning(_ context.Context, _, _ entity.ID) (bool, error) {
	return false, nil
}
