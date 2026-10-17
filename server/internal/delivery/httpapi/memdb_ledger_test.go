// Adapter ArtifactRepo + ArtifactStore, DecisionRepo, OrderRepo, LedgerRepo
// di atas memStore (E2E langkah 29).
package httpapi

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ---------- ArtifactRepo ----------

type memArtifacts struct{ s *memStore }

func (m memArtifacts) GetByID(_ context.Context, id entity.ID) (*entity.Artifact, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	a, ok := m.s.artifacts[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (m memArtifacts) ListByRun(_ context.Context, runID entity.ID) ([]entity.Artifact, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.Artifact
	for _, a := range m.s.artifacts {
		if a.RunID == runID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m memArtifacts) ListLatestByProject(_ context.Context, projectID entity.ID) ([]entity.Artifact, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	latest := map[entity.Stage]entity.Artifact{}
	for _, a := range m.s.artifacts {
		run, ok := m.s.runs[a.RunID]
		if !ok || run.ProjectID != projectID {
			continue
		}
		// created_at terbaru menang (meniru distinct on (kind) order by desc).
		if prev, ok := latest[a.Kind]; !ok || a.CreatedAt.After(prev.CreatedAt) {
			latest[a.Kind] = *a
		}
	}
	out := make([]entity.Artifact, 0, len(latest))
	for _, a := range latest {
		out = append(out, a)
	}
	return out, nil
}

func (m memArtifacts) Update(_ context.Context, a *entity.Artifact) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if _, ok := m.s.artifacts[a.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *a
	m.s.artifacts[a.ID] = &cp
	return nil
}

func (m memArtifacts) InsertVersion(_ context.Context, v *entity.ArtifactVersion) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, ex := range m.s.versions {
		if ex.ArtifactID == v.ArtifactID && ex.Version == v.Version {
			return port.ErrConflict
		}
	}
	cp := *v
	m.s.versions = append(m.s.versions, cp)
	return nil
}

// HasVersionSet cek (artifact_id, version) di history (uq_artifact_versions).
func (m memArtifacts) HasVersionSet(_ context.Context, artifactID entity.ID, version int) (bool, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, ex := range m.s.versions {
		if ex.ArtifactID == artifactID && ex.Version == version {
			return true, nil
		}
	}
	return false, nil
}

func (m memArtifacts) MarkDownstreamStale(_ context.Context, _ entity.ID, _ entity.Stage) error {
	return nil
}

// Upsert meniru uq_artifacts_run_kind (satu artifact per run+kind).
func (m memArtifacts) Upsert(_ context.Context, a *entity.Artifact) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, ex := range m.s.artifacts {
		if ex.RunID == a.RunID && ex.Kind == a.Kind {
			a.ID = ex.ID
			a.Version = ex.Version + 1
			cp := *a
			m.s.artifacts[ex.ID] = &cp
			return nil
		}
	}
	cp := *a
	m.s.artifacts[a.ID] = &cp
	return nil
}

// ---------- DecisionRepo ----------

type memDecisions struct{ s *memStore }

func (m memDecisions) Create(_ context.Context, d *entity.Decision) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	cp := *d
	m.s.decisions = append(m.s.decisions, cp)
	return nil
}

func (m memDecisions) ListByRun(_ context.Context, runID entity.ID) ([]entity.Decision, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.Decision
	for _, x := range m.s.decisions {
		if x.RunID == runID {
			out = append(out, x)
		}
	}
	return out, nil
}

func (m memDecisions) ListAll(_ context.Context, page port.Page) ([]entity.Decision, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	out := append([]entity.Decision(nil), m.s.decisions...)
	return pageSlice(out, page), nil
}

func (m memDecisions) NextRound(_ context.Context, _ entity.ID, _ entity.Stage) (int, error) {
	return 1, nil
}

// ---------- OrderRepo ----------

type memOrders struct{ s *memStore }

func (m memOrders) Create(_ context.Context, o *entity.Order) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	cp := *o
	m.s.orders[o.ID] = &cp
	return nil
}

func (m memOrders) GetByID(_ context.Context, id entity.ID) (*entity.Order, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	o, ok := m.s.orders[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *o
	return &cp, nil
}

func (m memOrders) GetByInvoiceNumber(_ context.Context, inv string) (*entity.Order, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, o := range m.s.orders {
		if o.InvoiceNumber == inv {
			cp := *o
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (m memOrders) GetByRequestID(_ context.Context, req string) (*entity.Order, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, o := range m.s.orders {
		if o.RequestID == req {
			cp := *o
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (m memOrders) Update(_ context.Context, o *entity.Order) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if _, ok := m.s.orders[o.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *o
	m.s.orders[o.ID] = &cp
	return nil
}

// ---------- LedgerRepo ----------

type memLedger struct{ s *memStore }

func (m memLedger) entry(e LedgerEntryRef) { m.s.ledger = append(m.s.ledger, e) }

func (m memLedger) ChargeForRun(_ context.Context, uid entity.ID, run *entity.PlanRun) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.profiles[uid]
	if !ok {
		return port.ErrNotFound
	}
	if p.Credits < run.CreditsCharged {
		return port.ErrInsufficientCredits
	}
	// uq_plan_runs_one_running: run running lain di project sama = konflik.
	for _, r := range m.s.runs {
		if r.ProjectID == run.ProjectID && r.Status == entity.RunRunning {
			return port.ErrConflict
		}
	}
	p.Credits -= run.CreditsCharged
	cp := *run
	m.s.runs[run.ID] = &cp
	m.entry(LedgerEntryRef{User: uid, Delta: -run.CreditsCharged, Reason: "plan_run",
		RefType: "plan_run", RefID: run.ID})
	return nil
}

func (m memLedger) ChargeForRegenerate(_ context.Context, uid, artID entity.ID, credits int) (entity.ID, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.profiles[uid]
	if !ok {
		return "", port.ErrNotFound
	}
	if p.Credits < credits {
		return "", port.ErrInsufficientCredits
	}
	p.Credits -= credits
	id := entity.ID("0197a23e-9999-7000-8000-0000000000c1")
	m.entry(LedgerEntryRef{User: uid, Delta: -credits, Reason: "regenerate",
		RefType: "charge", RefID: artID})
	return id, nil
}

// RefundRun idempoten: refund kedua untuk run sama = no-op (unique index DB).
func (m memLedger) RefundRun(_ context.Context, run *entity.PlanRun, credits int) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, e := range m.s.ledger {
		if e.RefType == "refund_run" && e.RefID == run.ID {
			return nil
		}
	}
	if p, ok := m.s.profiles[run.UserID]; ok {
		p.Credits += credits
	}
	m.entry(LedgerEntryRef{User: run.UserID, Delta: credits, Reason: "refund",
		RefType: "refund_run", RefID: run.ID})
	return nil
}

func (m memLedger) RefundCharge(_ context.Context, uid, chargeID entity.ID, credits int) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, e := range m.s.ledger {
		if e.RefType == "refund_charge" && e.RefID == chargeID {
			return nil
		}
	}
	if p, ok := m.s.profiles[uid]; ok {
		p.Credits += credits
	}
	m.entry(LedgerEntryRef{User: uid, Delta: credits, Reason: "refund",
		RefType: "refund_charge", RefID: chargeID})
	return nil
}

func (m memLedger) Grant(_ context.Context, uid entity.ID, credits int, actor entity.ID, key string) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, e := range m.s.ledger {
		if e.RefType == "grant" && e.Key == key {
			return nil
		}
	}
	p, ok := m.s.profiles[uid]
	if !ok {
		return port.ErrNotFound
	}
	p.Credits += credits
	m.entry(LedgerEntryRef{User: uid, Delta: credits, Reason: "grant",
		RefType: "grant", RefID: actor, Key: key})
	return nil
}

// SettleTopUp idempoten per order (unique index uq_ledger_one_topup_per_order).
// Update baris order (status, paid_at) seperti SQL asli, lalu kredit masuk.
func (m memLedger) SettleTopUp(_ context.Context, o *entity.Order) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, e := range m.s.ledger {
		if e.RefType == "topup" && e.RefID == o.ID {
			return nil
		}
	}
	stored, ok := m.s.orders[o.ID]
	if !ok {
		return port.ErrNotFound
	}
	if stored.Status != entity.OrderPending && stored.Status != entity.OrderExpire {
		return nil
	}
	p, ok := m.s.profiles[o.UserID]
	if !ok {
		return port.ErrNotFound
	}
	now := time.Now().UTC()
	stored.Status = entity.OrderSettlement
	stored.PaidAt = &now
	p.Credits += stored.CreditsGranted
	m.entry(LedgerEntryRef{User: o.UserID, Delta: stored.CreditsGranted, Reason: "topup",
		RefType: "topup", RefID: o.ID})
	return nil
}
