package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// LedgerRepo memegang semua perubahan kredit. SETIAP perubahan dalam satu
// transaksi: SELECT ... FOR UPDATE pada profiles -> tulis ledger -> update
// saldo (plan/security.md bagian 7, auth_billing.md bagian 3).
type LedgerRepo struct{ db dbtx }

const insertLedgerSQL = `insert into credits_ledger
	(id, user_id, delta, reason, ref_type, ref_id, actor_id, idempotency_key)
	values ($1,$2,$3,$4,$5,$6,$7,$8)`

// lockCredits mengunci baris profiles dan mengembalikan saldo terkini.
func lockCredits(ctx context.Context, tx pgx.Tx, userID entity.ID) (int, error) {
	var credits int
	err := tx.QueryRow(ctx, `select credits from profiles where id=$1 for update`, userID).Scan(&credits)
	if err != nil {
		return 0, mapErr(err)
	}
	return credits, nil
}

// addCredits menulis saldo baru. Pelanggaran CHECK credits >= 0 (23514)
// dikembalikan sebagai insufficient_credits (402).
func addCredits(ctx context.Context, tx pgx.Tx, userID entity.ID, newBalance int) error {
	_, err := tx.Exec(ctx, `update profiles set credits=$2 where id=$1`, userID, newBalance)
	if err != nil {
		if isCheck(err) {
			return port.ErrInsufficientCredits
		}
		return mapErr(err)
	}
	return nil
}

// ChargeForRun memotong kredit untuk run baru dalam SATU transaksi
// (auth_billing.md 3.1): FOR UPDATE -> cek saldo -> insert plan_runs ->
// insert ledger (-credits, reason run, ref plan_run) -> update saldo -> commit.
func (r *LedgerRepo) ChargeForRun(ctx context.Context, userID entity.ID, run *entity.PlanRun) error {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck // rollback setelah commit = no-op

	credits, err := lockCredits(ctx, tx, userID)
	if err != nil {
		return err
	}
	if credits < 1 {
		return port.ErrInsufficientCredits
	}

	if _, err := tx.Exec(ctx,
		`insert into plan_runs (id, project_id, user_id, status, current_stage, progress, total,
		   escalations, credits_charged, model, tokens_in, tokens_out, cost, error,
		   started_at, finished_at, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		run.ID, run.ProjectID, run.UserID, run.Status, run.CurrentStage, run.Progress,
		run.Total, run.Escalations, run.CreditsCharged, run.Model, run.TokensIn,
		run.TokensOut, run.Cost, run.Error, run.StartedAt, run.FinishedAt, run.CreatedAt); err != nil {
		if isUnique(err) {
			// uq_plan_runs_one_running: sudah ada run running di project ini
			return port.ErrConflict
		}
		return mapErr(err)
	}

	if _, err := tx.Exec(ctx, insertLedgerSQL,
		run.ID, userID, -run.CreditsCharged, entity.ReasonRun, entity.RefPlanRun, run.ID, nil, nil); err != nil {
		return mapErr(err)
	}
	if err := addCredits(ctx, tx, userID, credits-run.CreditsCharged); err != nil {
		return err
	}
	return mapErr(tx.Commit(ctx))
}

// ChargeForRegenerate memotong kredit untuk satu tahap (auth_billing.md 3.4)
// dan mengembalikan id baris charge untuk refund per baris.
func (r *LedgerRepo) ChargeForRegenerate(ctx context.Context, userID entity.ID, artifactID entity.ID, charged int) (entity.ID, error) {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return "", mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	credits, err := lockCredits(ctx, tx, userID)
	if err != nil {
		return "", err
	}
	if credits < charged {
		return "", port.ErrInsufficientCredits
	}
	chargeID := entity.ID(newUUID())
	if _, err := tx.Exec(ctx, insertLedgerSQL,
		chargeID, userID, -charged, entity.ReasonRegenerate, entity.RefArtifact, artifactID, nil, nil); err != nil {
		return "", mapErr(err)
	}
	if err := addCredits(ctx, tx, userID, credits-charged); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", mapErr(err)
	}
	return chargeID, nil
}

// RefundRun mengembalikan kredit run gagal. Sekali per run:
// uq_ledger_one_refund_per_run; konflik = refund sudah ada, bukan error.
func (r *LedgerRepo) RefundRun(ctx context.Context, run *entity.PlanRun, credits int) error {
	return r.refundRef(ctx, run.UserID, credits, entity.RefPlanRun, run.ID)
}

// RefundCharge mengembalikan satu baris charge regenerate
// (uq_ledger_one_refund_per_charge).
func (r *LedgerRepo) RefundCharge(ctx context.Context, userID entity.ID, chargeID entity.ID, credits int) error {
	return r.refundRef(ctx, userID, credits, entity.RefLedger, chargeID)
}

func (r *LedgerRepo) refundRef(ctx context.Context, userID entity.ID, credits int, refType entity.LedgerRefType, refID entity.ID) error {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	if _, err := tx.Exec(ctx, insertLedgerSQL,
		newUUID(), userID, credits, entity.ReasonRefund, refType, refID, nil, nil); err != nil {
		if isUnique(err) {
			return nil // refund untuk ref ini sudah ada
		}
		return mapErr(err)
	}
	cur, err := lockCredits(ctx, tx, userID)
	if err != nil {
		return err
	}
	if err := addCredits(ctx, tx, userID, cur+credits); err != nil {
		return err
	}
	return mapErr(tx.Commit(ctx))
}

// Grant menambah kredit oleh admin. Idempoten lewat idempotency_key
// (uq_ledger_one_grant_per_key). Replay = no-op, bukan error.
func (r *LedgerRepo) Grant(ctx context.Context, userID entity.ID, credits int, actorID entity.ID, idempotencyKey string) error {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	if _, err := tx.Exec(ctx, insertLedgerSQL,
		newUUID(), userID, credits, entity.ReasonGrant, entity.RefNone, nil, actorID, idempotencyKey); err != nil {
		if isUnique(err) {
			return nil // kunci idempotensi sudah dipakai
		}
		return mapErr(err)
	}
	cur, err := lockCredits(ctx, tx, userID)
	if err != nil {
		return err
	}
	if err := addCredits(ctx, tx, userID, cur+credits); err != nil {
		return err
	}
	return mapErr(tx.Commit(ctx))
}

// SettleTopUp men-settle order topup dalam satu transaksi
// (security.md bagian 6 langkah 9):
// UPDATE ... WHERE status='pending' (RowsAffected 0 = TIDAK ADA grant) ->
// insert ledger (+credits_granted, reason topup, ref order) -> update saldo.
// Jaring terakhir: uq_ledger_one_topup_per_order.
func (r *LedgerRepo) SettleTopUp(ctx context.Context, o *entity.Order) error {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`update orders set status='settlement', paid_at=now(), updated_at=now()
		 where id=$1 and status='pending'`, o.ID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		// bukan pending: settlement/expire. Webhook replay harus 200, jadi
		// konflik di sini dibaca usecase sebagai "sudah diproses".
		return port.ErrConflict
	}
	if _, err := tx.Exec(ctx, insertLedgerSQL,
		newUUID(), o.UserID, o.CreditsGranted, entity.ReasonTopUp, entity.RefOrder, o.ID, nil, nil); err != nil {
		if isUnique(err) {
			return port.ErrConflict
		}
		return mapErr(err)
	}
	cur, err := lockCredits(ctx, tx, o.UserID)
	if err != nil {
		return err
	}
	if err := addCredits(ctx, tx, o.UserID, cur+o.CreditsGranted); err != nil {
		return err
	}
	return mapErr(tx.Commit(ctx))
}
