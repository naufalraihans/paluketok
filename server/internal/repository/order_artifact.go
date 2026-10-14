package repository

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// --- OrderRepo ---

type OrderRepo struct{ db dbtx }

const orderCols = `id, user_id, package_id, invoice_number, amount_idr, credits_granted,
	status, request_id, payment_url, expired_date, paid_at, created_at, updated_at`

func scanOrder(row scannable) (*entity.Order, error) {
	var o entity.Order
	if err := row.Scan(&o.ID, &o.UserID, &o.PackageID, &o.InvoiceNumber, &o.AmountIDR,
		&o.CreditsGranted, &o.Status, &o.RequestID, &o.PaymentURL, &o.ExpiredDate,
		&o.PaidAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &o, nil
}

func (r *OrderRepo) Create(ctx context.Context, o *entity.Order) error {
	_, err := r.db.Exec(ctx,
		`insert into orders (id, user_id, package_id, invoice_number, amount_idr,
		   credits_granted, status, request_id, payment_url, expired_date, paid_at,
		   created_at, updated_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		o.ID, o.UserID, o.PackageID, o.InvoiceNumber, o.AmountIDR, o.CreditsGranted,
		o.Status, o.RequestID, o.PaymentURL, o.ExpiredDate, o.PaidAt, o.CreatedAt, o.UpdatedAt)
	return mapErr(err)
}

func (r *OrderRepo) GetByID(ctx context.Context, id entity.ID) (*entity.Order, error) {
	return scanOrder(r.db.QueryRow(ctx, `select `+orderCols+` from orders where id=$1`, id))
}

func (r *OrderRepo) GetByInvoiceNumber(ctx context.Context, inv string) (*entity.Order, error) {
	return scanOrder(r.db.QueryRow(ctx, `select `+orderCols+` from orders where invoice_number=$1`, inv))
}

func (r *OrderRepo) GetByRequestID(ctx context.Context, reqID string) (*entity.Order, error) {
	return scanOrder(r.db.QueryRow(ctx, `select `+orderCols+` from orders where request_id=$1`, reqID))
}

// Update menulis ulang kolom yang Diedit usecase (payment_url, expired_date).
// Transisi status settlement/expire TIDAK lewat sini: lewat LedgerRepo agar
// selalu guard WHERE status='pending'.
func (r *OrderRepo) Update(ctx context.Context, o *entity.Order) error {
	tag, err := r.db.Exec(ctx,
		`update orders set payment_url=$2, expired_date=$3, updated_at=$4 where id=$1`,
		o.ID, o.PaymentURL, o.ExpiredDate, time.Now())
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// --- ArtifactRepo ---

type ArtifactRepo struct{ db dbtx }

const artifactCols = `id, run_id, user_id, kind, content, version, status, created_at, updated_at`

func scanArtifact(row scannable) (*entity.Artifact, error) {
	var a entity.Artifact
	if err := row.Scan(&a.ID, &a.RunID, &a.UserID, &a.Kind, &a.Content, &a.Version,
		&a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &a, nil
}

func (r *ArtifactRepo) GetByID(ctx context.Context, id entity.ID) (*entity.Artifact, error) {
	return scanArtifact(r.db.QueryRow(ctx, `select `+artifactCols+` from artifacts where id=$1`, id))
}

func (r *ArtifactRepo) ListByRun(ctx context.Context, runID entity.ID) ([]entity.Artifact, error) {
	rows, err := r.db.Query(ctx, `select `+artifactCols+` from artifacts where run_id=$1 order by kind`, runID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, mapErr(rows.Err())
}

// Update menimpa konten/versi/status. Versi lama disimpan dulu lewat
// InsertVersion oleh usecase (erd.md aturan versi). Insert baru bila belum ada
// (upsert pada unique (run_id, kind)).
func (r *ArtifactRepo) Update(ctx context.Context, a *entity.Artifact) error {
	_, err := r.db.Exec(ctx,
		`insert into artifacts (id, run_id, user_id, kind, content, version, status, created_at, updated_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 on conflict (run_id, kind) do update set
		   content = excluded.content, version = excluded.version,
		   status = excluded.status, updated_at = excluded.updated_at`,
		a.ID, a.RunID, a.UserID, a.Kind, a.Content, a.Version, a.Status, a.CreatedAt, a.UpdatedAt)
	return mapErr(err)
}

func (r *ArtifactRepo) InsertVersion(ctx context.Context, v *entity.ArtifactVersion) error {
	_, err := r.db.Exec(ctx,
		`insert into artifact_versions (id, artifact_id, version, content, created_by_run_id, created_at)
		 values ($1,$2,$3,$4,$5,$6)`,
		v.ID, v.ArtifactID, v.Version, v.Content, v.CreatedByRunID, v.CreatedAt)
	return mapErr(err)
}

// HasVersionSet true bila (artifact_id, version) sudah ada di history
// (uq_artifact_versions). Dipakai regenerate agar versi lama hasil Upsert
// pipeline tidak ditulis ulang (23505).
func (r *ArtifactRepo) HasVersionSet(ctx context.Context, artifactID entity.ID, version int) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`select exists (select 1 from artifact_versions where artifact_id=$1 and version=$2)`,
		artifactID, version).Scan(&ok)
	return ok, mapErr(err)
}

// --- DecisionRepo ---

type DecisionRepo struct{ db dbtx }

func (r *DecisionRepo) Create(ctx context.Context, d *entity.Decision) error {
	_, err := r.db.Exec(ctx,
		`insert into decisions (id, run_id, user_id, stage, question, state, criteria, gate,
		   round, choice, confidence, probabilities, jev_decision_id, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		d.ID, d.RunID, d.UserID, d.Stage, d.Question, d.State, d.Criteria, d.Gate,
		d.Round, d.Choice, d.Confidence, d.Probabilities, d.JEVDecisionID, d.CreatedAt)
	return mapErr(err)
}

func (r *DecisionRepo) ListByRun(ctx context.Context, runID entity.ID) ([]entity.Decision, error) {
	rows, err := r.db.Query(ctx,
		`select id, run_id, user_id, stage, question, state, criteria, gate, round,
		        choice, confidence, probabilities, jev_decision_id, created_at
		   from decisions where run_id=$1 order by created_at, round`, runID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.Decision
	for rows.Next() {
		var d entity.Decision
		if err := rows.Scan(&d.ID, &d.RunID, &d.UserID, &d.Stage, &d.Question, &d.State,
			&d.Criteria, &d.Gate, &d.Round, &d.Choice, &d.Confidence, &d.Probabilities,
			&d.JEVDecisionID, &d.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, d)
	}
	return out, mapErr(rows.Err())
}

func (r *DecisionRepo) NextRound(ctx context.Context, runID entity.ID, stage entity.Stage) (int, error) {
	var n *int
	err := r.db.QueryRow(ctx,
		`select max(round) from decisions where run_id=$1 and stage=$2`, runID, stage).Scan(&n)
	if err != nil {
		return 0, mapErr(err)
	}
	if n == nil {
		return 1, nil
	}
	return *n + 1, nil
}
