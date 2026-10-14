// Method admin + regenerate pada repo yang sudah ada (workflow.md 26-29,
// openapi.md bagian admin, planner.md bagian 7).
package repository

import (
	"context"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// --- projects ---

func (r *ProjectRepo) ListAll(ctx context.Context, page port.Page) ([]entity.Project, error) {
	rows, err := r.db.Query(ctx,
		`select id, owner_id, name, brief, stack, status, created_at, updated_at
		 from projects order by created_at desc, id desc limit $1 offset $2`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.Project
	for rows.Next() {
		var p entity.Project
		var stack []byte
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Brief, &stack, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, mapErr(err)
		}
		if p.Stack, err = decodeStack(stack); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, mapErr(rows.Err())
}

// --- plan_runs ---

func (r *RunRepo) ListAllRuns(ctx context.Context, status string, userID entity.ID, page port.Page) ([]entity.PlanRun, error) {
	// user_id kosong = tanpa filter. String kosong di-cast ke uuid → 22P02,
	// jadi kirim NULL saat kosong (pola `$2::uuid is null`).
	var uid any
	if userID != "" {
		uid = string(userID)
	}
	rows, err := r.db.Query(ctx, `
		select `+runCols+` from plan_runs
		where ($1 = '' or status = $1) and ($2::uuid is null or user_id = $2)
		order by created_at desc limit $3 offset $4`,
		status, uid, page.Limit, page.Offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.PlanRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, mapErr(rows.Err())
}

// --- decisions ---

func (r *DecisionRepo) ListAll(ctx context.Context, page port.Page) ([]entity.Decision, error) {
	rows, err := r.db.Query(ctx,
		`select id, run_id, user_id, stage, question, state, criteria, gate, round,
		        choice, confidence, probabilities, jev_decision_id, created_at
		 from decisions order by created_at desc limit $1 offset $2`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.Decision
	for rows.Next() {
		var d entity.Decision
		if err := rows.Scan(&d.ID, &d.RunID, &d.UserID, &d.Stage, &d.Question, &d.State,
			&d.Criteria, &d.Gate, &d.Round, &d.Choice, &d.Confidence,
			&d.Probabilities, &d.JEVDecisionID, &d.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, d)
	}
	return out, mapErr(rows.Err())
}

// --- profiles ---

// SetPlanTier mengubah plan_tier. Kolom is_admin sengaja tidak tersentuh:
// tak ada jalur API yang mengubah is_admin (openapi.md aturan akhir).
func (r *ProfileRepo) SetPlanTier(ctx context.Context, id entity.ID, tier entity.PlanTier) error {
	tag, err := r.db.Exec(ctx,
		`update profiles set plan_tier=$2 where id=$1`, id, tier)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// --- artifacts (regenerate, planner.md bagian 7) ---

// MarkDownstreamStale menandai semua artefak tahap SESUDAH `after` di run
// yang sama menjadi 'stale'.
func (r *ArtifactRepo) MarkDownstreamStale(ctx context.Context, runID entity.ID, after entity.Stage) error {
	_, err := r.db.Exec(ctx,
		`update artifacts set status='stale', updated_at=now()
		 where run_id=$1 and kind = any($2)`,
		runID, stagesAfter(after))
	return mapErr(err)
}

// stagesAfter mengembalikan kolom entity.Stages sesudah `after`. NULL bila
// after tahap terakhir (tidak ada downstream).
func stagesAfter(after entity.Stage) []entity.Stage {
	var out []entity.Stage
	found := false
	for _, s := range entity.Stages {
		if found {
			out = append(out, s)
		}
		if s == after {
			found = true
		}
	}
	return out
}
