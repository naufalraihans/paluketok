// Adapter antara repository.Repos (konkret) dan kontrak worker.RunStore.
// Dipisah supaya internal/worker tidak bergantung pada paket repository.
package main

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/repository"
)

type workerRunAdapter struct{ repos *repository.Repos }

func (a workerRunAdapter) GetFull(ctx context.Context, id entity.ID) (*entity.PlanRun, error) {
	return a.repos.Runs.GetByID(ctx, id)
}

func (a workerRunAdapter) ClaimQueued(ctx context.Context) (*port.QueuedRun, error) {
	return a.repos.Runs.ClaimQueued(ctx)
}

func (a workerRunAdapter) SaveProgress(ctx context.Context, id entity.ID, stage entity.Stage, progress int, status entity.RunStatus) error {
	return a.repos.Runs.PatchProgress(ctx, id, stage, progress, status)
}

func (a workerRunAdapter) SaveTotal(ctx context.Context, id entity.ID, total int) error {
	return a.repos.Runs.PatchTotal(ctx, id, total)
}

// workerPlanStageAdapter menyambungkan plan_stages ke kontrak worker.
// Dipisah seperti adapter run supaya internal/worker tidak mengenal repository.
type workerPlanStageAdapter struct{ repos *repository.Repos }

func (a workerPlanStageAdapter) ListByRun(ctx context.Context, runID entity.ID) ([]entity.PlanStage, error) {
	return a.repos.PlanStages.ListByRun(ctx, runID)
}

func (a workerPlanStageAdapter) ReplaceByRun(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error {
	return a.repos.PlanStages.ReplaceByRun(ctx, runID, stages)
}

func (a workerRunAdapter) Finish(ctx context.Context, id entity.ID, status entity.RunStatus, runErr string, now time.Time) error {
	return a.repos.Runs.PatchFinished(ctx, id, status, runErr, now)
}

func (a workerRunAdapter) ListQueued(ctx context.Context) ([]port.QueuedRun, error) {
	return a.repos.Runs.ListQueued(ctx)
}

func (a workerRunAdapter) RequeueRunning(ctx context.Context, now time.Time) (int, error) {
	return a.repos.Runs.RequeueRunning(ctx, now)
}

func (a workerRunAdapter) HasOtherRunning(ctx context.Context, projectID, excludeID entity.ID) (bool, error) {
	return a.repos.Runs.HasOtherRunning(ctx, projectID, excludeID)
}
