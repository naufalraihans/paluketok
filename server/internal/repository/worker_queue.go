// Operasi antrean worker untuk plan_runs (plan/workflow.md 22, 25;
// plan/security.md bagian 11). Klaim atomik dengan FOR UPDATE SKIP LOCKED:
// dua worker tidak pernah mendapat run yang sama.
package repository

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ClaimQueued mengklaim satu run 'queued' -> 'running' atomik.
// SKIP LOCKED membuat dua claims bersamaan melompati baris yang sudah
// dikunci worker lain, jadi satu run hanya diambil satu worker.
func (r *RunRepo) ClaimQueued(ctx context.Context) (*port.QueuedRun, error) {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	row := tx.QueryRow(ctx, `
		update plan_runs set status='running', started_at=coalesce(started_at, now())
		where id = (
			select id from plan_runs
			where status='queued'
			order by created_at
			limit 1
			for update skip locked
		)
		returning id, project_id, user_id`)
	var q port.QueuedRun
	if err := row.Scan(&q.ID, &q.ProjectID, &q.UserID); err != nil {
		return nil, mapErr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, mapErr(err)
	}
	return &q, nil
}

// ListQueued mengembalikan run 'queued' urut created_at untuk pemanasan
// antrean in-memory saat start (plan/security.md bagian 11).
func (r *RunRepo) ListQueued(ctx context.Context) ([]port.QueuedRun, error) {
	rows, err := r.db.Query(ctx, `
		select id, project_id, user_id from plan_runs
		where status='queued' order by created_at`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []port.QueuedRun
	for rows.Next() {
		var q port.QueuedRun
		if err := rows.Scan(&q.ID, &q.ProjectID, &q.UserID); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, q)
	}
	return out, mapErr(rows.Err())
}

// RequeueRunning menandai semua run 'running' sisa proses mati sebagai failed
// dengan alasan server_restart dan me-refund kreditnya sekali
// (uq_ledger_one_refund_per_run membuat refund kedua menjadi no-op).
func (r *RunRepo) RequeueRunning(ctx context.Context, now time.Time) (int, error) {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	rows, err := tx.Query(ctx, `
		update plan_runs
		set status='failed', error='server_restart', finished_at=$1
		where status='running'
		returning id, user_id, credits_charged`, now)
	if err != nil {
		return 0, mapErr(err)
	}
	type dead struct {
		id      entity.ID
		userID  entity.ID
		credits int
	}
	var deads []dead
	for rows.Next() {
		var d dead
		if err := rows.Scan(&d.id, &d.userID, &d.credits); err != nil {
			rows.Close()
			return 0, mapErr(err)
		}
		deads = append(deads, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, mapErr(err)
	}

	for _, d := range deads {
		// Refund idempoten: konflik uq_ledger_one_refund_per_run diabaikan.
		if _, err := tx.Exec(ctx, insertLedgerSQL,
			newUUID(), d.userID, d.credits, entity.ReasonRefund, entity.RefPlanRun, d.id, nil, nil); err != nil {
			if !isUnique(err) {
				return 0, mapErr(err)
			}
		}
		cur, err := lockCredits(ctx, tx, d.userID)
		if err != nil {
			return 0, err
		}
		if err := addCredits(ctx, tx, d.userID, cur+d.credits); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, mapErr(err)
	}
	return len(deads), nil
}
