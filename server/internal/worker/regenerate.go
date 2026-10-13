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

	draft, err := p.generateWithRetry(ctx, run, art.Kind)
	if err != nil {
		return "", err
	}
	if err := validate(art.Kind, draft, prev); err != nil {
		return "", fmt.Errorf("validasi %s: %w", art.Kind, err)
	}
	return draft, nil
}

// artifactDigestMap: digest ringkas per tahap untuk prompt.
func artifactDigestMap(m map[entity.Stage]string) map[entity.Stage]string {
	out := make(map[entity.Stage]string, len(m))
	for k, v := range m {
		out[k] = truncate(v, 400)
	}
	return out
}
