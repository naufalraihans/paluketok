// Titik sambung skema tahap ke usecase.
//
// Pipeline memenuhi usecase.StageProposer dan usecase.StageStore supaya cmd/web
// cukup memasang satu objek: pipe.SetStageProposer(pipe, pipe).
package worker

import (
	"context"

	"paluketok/server/internal/entity"
)

// ProposeStages memenuhi usecase.StageProposer: menyusun usulan skema tahap
// untuk layar konfirmasi sebelum run dibuat.
func (p *Pipeline) ProposeStages(ctx context.Context, brief string, stack map[string]string) ([]entity.PlanStage, string) {
	cctx, cancel := context.WithTimeout(ctx, judgeTimeout)
	defer cancel()
	return ProposePlanStages(cctx, p.judge, brief, stack)
}

// ReplaceByRun memenuhi usecase.StageStore: menulis skema satu run.
//
// Skema run baru selalu kosong, jadi "replace" cukup: hapus sisa baris run ini
// lalu sisipkan yang baru. Satu transaksi supaya tidak pernah ada run dengan
// skema separuh.
func (p *Pipeline) ReplaceByRun(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error {
	if p.stages == nil {
		return nil
	}
	return p.stages.ReplaceByRun(ctx, runID, stages)
}

// ListByRun memenuhi usecase.StageStore.
func (p *Pipeline) ListByRun(ctx context.Context, runID entity.ID) ([]entity.PlanStage, error) {
	if p.stages == nil {
		return entity.DefaultPlanStages(runID), nil
	}
	return p.stages.ListByRun(ctx, runID)
}
