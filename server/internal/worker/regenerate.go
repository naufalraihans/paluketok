// RegenerateStage: generate ulang SATU tahap untuk POST /v1/artifacts/{id}/regenerate
// (planner.md bagian 7). Dipanggil usecase lewat kontrak Regenerator; konteks
// sinkron (request HTTP). Jeva TIDAK dipanggil ulang di jalur regenerate.
package worker

import (
	"context"
	"errors"
	"fmt"

	"paluketok/server/internal/entity"
)

// ErrGateEscalate: tahap ditolak gate ESCALATE; permintaan regenerate gagal.
var ErrGateEscalate = errors.New("regenerate: gate ESCALATE")

// RegenerateStage memenuhi usecase.Regenerator. Konteks artefak run asli
// dimuat ulang dari DB supaya prompt memakai artefak sesudahnya yang benar.
// Output TIDAK di upsert di sini: usecase menyimpan versi + tandai hilir
// (planner.md bagian 7 urutan versi).
func (p *Pipeline) RegenerateStage(ctx context.Context, art *entity.Artifact) (string, error) {
	run, err := p.runs.GetFull(ctx, art.RunID)
	if err != nil {
		return "", fmt.Errorf("run: %w", err)
	}
	// Muat ulang artefak run sebagai konteks prompt (urutan tahap tetap).
	prev := map[entity.Stage]string{}
	list, err := p.artifacts.ListByRun(ctx, run.ID)
	if err != nil {
		return "", fmt.Errorf("artifacts: %w", err)
	}
	for _, a := range list {
		if a.Kind != art.Kind {
			prev[a.Kind] = a.Content
		}
	}
	p.prev = prev
	p.prevDigest = artifactDigestMap(prev)

	// Skema tahap run dipakai supaya regenerate memakai intent + output_format
	// yang benar. Run lama tanpa skema jatuh ke susunan awal, lalu tahap yang
	// diminta dicari dari sana.
	plan, err := p.planStages(ctx, run)
	if err != nil {
		return "", fmt.Errorf("skema tahap: %w", err)
	}
	st, ok := findPlanStage(plan, art.Kind)
	if !ok {
		return "", fmt.Errorf("tahap %s tidak ada di skema run", art.Kind)
	}

	draft, err := p.generateWithRetry(ctx, run, st)
	if err != nil {
		return "", err
	}
	if err := validatePlanStage(st, draft, prev); err != nil {
		return "", fmt.Errorf("validasi %s: %w", art.Kind, err)
	}
	return draft, nil
}

// findPlanStage mencari satu tahap di skema run.
func findPlanStage(plan []entity.PlanStage, key entity.Stage) (entity.PlanStage, bool) {
	for _, st := range plan {
		if st.Key == key {
			return st, true
		}
	}
	return entity.PlanStage{}, false
}

// artifactDigestMap: digest ringkas per tahap untuk prompt.
func artifactDigestMap(m map[entity.Stage]string) map[entity.Stage]string {
	out := make(map[entity.Stage]string, len(m))
	for k, v := range m {
		out[k] = truncate(v, 400)
	}
	return out
}
