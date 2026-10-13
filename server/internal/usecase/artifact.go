// Baca satu artefak + regenerate: usecase tipis di atas RunUsecase deps
// (openapi.md baris 20-21; aturan regenerate planner.md bagian 7).
package usecase

import (
	"context"
	"errors"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// GetArtifact satu artefak jika actor owner run-nya (atau admin).
func (u *RunUsecase) GetArtifact(ctx context.Context, actor entity.Actor, artifactID entity.ID) (*entity.Artifact, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	if !entity.ValidID(string(artifactID)) {
		return nil, ErrNotFound
	}
	art, err := u.deps.Artifacts.GetByID(ctx, artifactID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	run, err := u.deps.Runs.GetByID(ctx, art.RunID)
	if err != nil {
		return nil, wrap(err)
	}
	if run.UserID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return art, nil
}
