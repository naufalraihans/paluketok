// Upsert artifact pertama per tahap (docs/plan/workflow.md 23): insert artifacts
// + artifact_versions versi 1. Konflik uq_artifacts_run_kind (run diulang
// setelah restart sebelum sempat selesai) = update konten, versi tetap 1.
package repository

import (
	"context"

	"paluketok/server/internal/entity"
)

// Upsert menulis artifacts dan versi pertamanya dalam satu transaksi.
func (r *ArtifactRepo) Upsert(ctx context.Context, a *entity.Artifact) error {
	tx, err := r.db.(beginner).Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`insert into artifacts (id, run_id, user_id, kind, content, version, status, created_at, updated_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 on conflict (run_id, kind) do update
		 set content = excluded.content, updated_at = excluded.updated_at`,
		a.ID, a.RunID, a.UserID, a.Kind, a.Content, a.Version, a.Status, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return mapErr(err)
	}
	// Versi 1 hanya saat artefak baru; regenerate (F lanjutan) menambah versi.
	if _, err := tx.Exec(ctx,
		`insert into artifact_versions (id, artifact_id, version, content, created_by_run_id, created_at)
		 select $1, id, $2, $3, $4, $5 from artifacts where run_id=$6 and kind=$7
		 and not exists (select 1 from artifact_versions v where v.artifact_id = artifacts.id and v.version = $2)`,
		newUUID(), a.Version, a.Content, a.RunID, a.UpdatedAt, a.RunID, a.Kind); err != nil {
		return mapErr(err)
	}
	return tx.Commit(ctx)
}
