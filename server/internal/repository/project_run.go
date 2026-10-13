package repository

import (
	"context"
	"encoding/json"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// encodeStack menyandikan entity.Stack ke jsonb. Kosong -> NULL.
func encodeStack(s entity.Stack) (any, error) {
	if s.IsZero() {
		return nil, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func decodeStack(raw []byte) (entity.Stack, error) {
	if len(raw) == 0 {
		return entity.Stack{}, nil
	}
	var s entity.Stack
	if err := json.Unmarshal(raw, &s); err != nil {
		return entity.Stack{}, err
	}
	return s, nil
}

// --- ProjectRepo ---

type ProjectRepo struct{ db dbtx }

func (r *ProjectRepo) GetByID(ctx context.Context, id entity.ID) (*entity.Project, error) {
	row := r.db.QueryRow(ctx,
		`select id, owner_id, name, brief, stack, status, created_at, updated_at
		 from projects where id = $1`, id)
	var p entity.Project
	var stack []byte
	err := row.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Brief, &stack, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	if p.Stack, err = decodeStack(stack); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) Create(ctx context.Context, p *entity.Project) error {
	stack, err := encodeStack(p.Stack)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`insert into projects (id, owner_id, name, brief, stack, status, created_at, updated_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.OwnerID, p.Name, p.Brief, stack, p.Status, p.CreatedAt, p.UpdatedAt)
	return mapErr(err)
}

func (r *ProjectRepo) ListByOwner(ctx context.Context, ownerID entity.ID, page port.Page) ([]entity.Project, error) {
	rows, err := r.db.Query(ctx,
		`select id, owner_id, name, brief, stack, status, created_at, updated_at
		 from projects where owner_id = $1 order by created_at desc, id desc limit $2 offset $3`,
		ownerID, page.Limit, page.Offset)
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

func (r *ProjectRepo) Update(ctx context.Context, p *entity.Project) error {
	stack, err := encodeStack(p.Stack)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx,
		`update projects set name=$2, brief=$3, stack=$4, status=$5, updated_at=$6
		 where id=$1 and owner_id=$7`,
		p.ID, p.Name, p.Brief, stack, p.Status, p.UpdatedAt, p.OwnerID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *ProjectRepo) Archive(ctx context.Context, id, ownerID entity.ID, now time.Time) error {
	tag, err := r.db.Exec(ctx,
		`update projects set status='archived', updated_at=$3 where id=$1 and owner_id=$2`,
		id, ownerID, now)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// --- RunRepo ---

type RunRepo struct{ db dbtx }

const runCols = `id, project_id, user_id, status, current_stage, progress, total,
	escalations, credits_charged, model, tokens_in, tokens_out, cost, error,
	started_at, finished_at, created_at`

func scanRun(row scannable) (*entity.PlanRun, error) {
	var run entity.PlanRun
	if err := row.Scan(&run.ID, &run.ProjectID, &run.UserID, &run.Status, &run.CurrentStage,
		&run.Progress, &run.Total, &run.Escalations, &run.CreditsCharged, &run.Model,
		&run.TokensIn, &run.TokensOut, &run.Cost, &run.Error, &run.StartedAt,
		&run.FinishedAt, &run.CreatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &run, nil
}

func (r *RunRepo) GetByID(ctx context.Context, id entity.ID) (*entity.PlanRun, error) {
	return scanRun(r.db.QueryRow(ctx, `select `+runCols+` from plan_runs where id=$1`, id))
}

func (r *RunRepo) ListByProject(ctx context.Context, projectID entity.ID) ([]entity.PlanRun, error) {
	rows, err := r.db.Query(ctx, `select `+runCols+` from plan_runs where project_id=$1 order by created_at desc`, projectID)
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

func (r *RunRepo) HasRunning(ctx context.Context, projectID entity.ID) (bool, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`select count(*) from plan_runs where project_id=$1 and status='running'`, projectID).Scan(&n)
	return n > 0, mapErr(err)
}

// HasOtherRunning sama tapi mengecualikan satu run (worker memeriksa
// saingan selain run yang sedang diproses; plan/planner.md bagian 8).
func (r *RunRepo) HasOtherRunning(ctx context.Context, projectID, excludeID entity.ID) (bool, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`select count(*) from plan_runs where project_id=$1 and status='running' and id<>$2`,
		projectID, excludeID).Scan(&n)
	return n > 0, mapErr(err)
}

func (r *RunRepo) Update(ctx context.Context, run *entity.PlanRun) error {
	tag, err := r.db.Exec(ctx,
		`update plan_runs set status=$2, current_stage=$3, progress=$4, escalations=$5,
		 model=$6, tokens_in=$7, tokens_out=$8, cost=$9, error=$10,
		 started_at=$11, finished_at=$12 where id=$1`,
		run.ID, run.Status, run.CurrentStage, run.Progress, run.Escalations, run.Model,
		run.TokensIn, run.TokensOut, run.Cost, run.Error, run.StartedAt, run.FinishedAt)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// PatchProgress memperbarui status + current_stage + progress saja
// (plan/planner.md bagian 1 langkah a0/f). Dipakai worker tiap tahap.
func (r *RunRepo) PatchProgress(ctx context.Context, id entity.ID, stage entity.Stage, progress int, status entity.RunStatus) error {
	tag, err := r.db.Exec(ctx,
		`update plan_runs set status=$2, current_stage=$3, progress=$4 where id=$1`,
		id, status, stage, progress)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// PatchFinished menutup run: status final + error + finished_at.
func (r *RunRepo) PatchFinished(ctx context.Context, id entity.ID, status entity.RunStatus, runErr string, now time.Time) error {
	tag, err := r.db.Exec(ctx,
		`update plan_runs set status=$2, error=$3, finished_at=$4 where id=$1`,
		id, status, nullIfEmpty(runErr), now)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

// nullIfEmpty: error kosong disimpan NULL, bukan string kosong.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
