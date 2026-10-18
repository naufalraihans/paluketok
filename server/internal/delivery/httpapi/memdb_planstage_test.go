// memPlanStages: skema tahap per run untuk test httpapi.
package httpapi

import (
	"context"

	"paluketok/server/internal/entity"
)

type memPlanStages struct{ s *memStore }

func (m memPlanStages) ListByRun(_ context.Context, runID entity.ID) ([]entity.PlanStage, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.PlanStage
	for _, p := range m.s.planStages {
		if p.RunID == runID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m memPlanStages) ReplaceByRun(_ context.Context, runID entity.ID, stages []entity.PlanStage) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	kept := m.s.planStages[:0]
	for _, p := range m.s.planStages {
		if p.RunID != runID {
			kept = append(kept, p)
		}
	}
	m.s.planStages = kept
	for _, p := range stages {
		p.RunID = runID
		m.s.planStages = append(m.s.planStages, p)
	}
	return nil
}
