package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"paluketok/server/internal/entity"
)

// PlanStageRepo akses plan_stages (migrasi 0007). Tabel ini tidak punya RLS
// policy untuk klien: hanya server (koneksi pemilik DB) yang membacanya.
type PlanStageRepo struct{ db *pgxpool.Pool }

// ReplaceByRun menulis skema satu run: hapus baris run ini, sisipkan yang baru.
// Satu transaksi supaya tidak pernah ada run dengan skema separuh.
func (r *PlanStageRepo) ReplaceByRun(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `delete from public.plan_stages where run_id=$1`, runID); err != nil {
		return mapErr(err)
	}
	for i := range stages {
		s := stages[i]
		id := s.ID
		if id == "" {
			id = entity.ID(newUUID())
		}
		pos := s.Position
		if pos == 0 {
			pos = i
		}
		var spec any
		if len(s.CheckSpec) > 0 {
			spec = []byte(s.CheckSpec)
		}
		if _, err := tx.Exec(ctx, `
			insert into public.plan_stages
				(id, run_id, key, label, position, intent, output_format, check_spec, depends_on)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, runID, string(s.Key), s.Label, pos, s.Intent,
			string(s.OutputFormat), spec, stageKeys(s.DependsOn)); err != nil {
			return mapErr(err)
		}
	}
	return mapErr(tx.Commit(ctx))
}

// stageKeys mengubah []Stage jadi []string untuk kolom text[].
func stageKeys(in []entity.Stage) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, string(s))
	}
	return out
}

// ListByRun mengembalikan skema tahap satu run, urut position.
func (r *PlanStageRepo) ListByRun(ctx context.Context, runID entity.ID) ([]entity.PlanStage, error) {
	rows, err := r.db.Query(ctx,
		`select id, run_id, key, label, position, intent, output_format, check_spec, depends_on, created_at
		 from plan_stages where run_id = $1 order by position, key`, runID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var out []entity.PlanStage
	for rows.Next() {
		var st entity.PlanStage
		var spec []byte
		var deps []string
		if err := rows.Scan(&st.ID, &st.RunID, &st.Key, &st.Label, &st.Position,
			&st.Intent, &st.OutputFormat, &spec, &deps, &st.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		if len(spec) > 0 {
			st.CheckSpec = json.RawMessage(spec)
		}
		for _, d := range deps {
			st.DependsOn = append(st.DependsOn, entity.Stage(d))
		}
		out = append(out, st)
	}
	return out, mapErr(rows.Err())
}

// ReplaceAll menulis skema tahap lengkap satu run: hapus lalu insert, dalam satu
// transaksi supaya pembaca tidak pernah melihat skema setengah jadi. ID dan
// CreatedAt diisi di sini (bukan oleh pemanggil) karena keduanya urusan kolom.
func (r *PlanStageRepo) ReplaceAll(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `delete from plan_stages where run_id = $1`, runID); err != nil {
		return mapErr(err)
	}
	now := time.Now().UTC()
	for _, st := range stages {
		var deps []string
		for _, d := range st.DependsOn {
			deps = append(deps, string(d))
		}
		var spec any
		if len(st.CheckSpec) > 0 {
			spec = []byte(st.CheckSpec)
		}
		id := st.ID
		if id == "" {
			id = entity.ID(newUUID())
		}
		created := st.CreatedAt
		if created.IsZero() {
			created = now
		}
		if _, err := tx.Exec(ctx,
			`insert into plan_stages
			   (id, run_id, key, label, position, intent, output_format, check_spec, depends_on, created_at)
			 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 on conflict (run_id, key) do update set
			   label = excluded.label,
			   position = excluded.position,
			   intent = excluded.intent,
			   output_format = excluded.output_format,
			   check_spec = excluded.check_spec,
			   depends_on = excluded.depends_on`,
			id, runID, st.Key, st.Label, st.Position, st.Intent, st.OutputFormat, spec, deps, created); err != nil {
			return mapErr(err)
		}
	}
	return mapErr(tx.Commit(ctx))
}
