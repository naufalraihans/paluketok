// Regenerate satu tahap (plan/planner.md bagian 7, auth_billing.md 3.4):
// 1 kredit di muka (ChargeForRegenerate ref artifact), generate ulang HANYA
// tahap itu, simpan versi baru + perbarui artifacts, tandai hilir stale,
// refund lewat RefundCharge bila gagal (idempoten per charge_id).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Regenerator satu tahap; implementasi nyata di worker (F6). Pemisahan
// supaya usecase tidak mengimpor worker.
type Regenerator interface {
	RegenerateStage(ctx context.Context, artifact *entity.Artifact) (string, error)
}

// RunUsecase menerima regenerator opsional (nil = regenerate 503).
func (u *RunUsecase) SetRegenerator(r Regenerator) { u.regenerator = r }

// RegenerateArtifact jalur POST /v1/artifacts/{id}/regenerate.
func (u *RunUsecase) RegenerateArtifact(ctx context.Context, actor entity.Actor, artifactID entity.ID) (*entity.Artifact, error) {
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
	if art.UserID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	run, err := u.deps.Runs.GetByID(ctx, art.RunID)
	if err != nil {
		return nil, wrap(err)
	}
	// Syarat: run induk done DAN tidak ada run running di project sama,
	// selain itu 422 (openapi.md baris 21).
	if run.Status != entity.RunDone {
		return nil, fmt.Errorf("%w: run induk %s", ErrValidation, run.Status)
	}
	other, err := u.deps.Runs.HasOtherRunning(ctx, run.ProjectID, run.ID)
	if err != nil {
		return nil, wrap(err)
	}
	if other {
		return nil, fmt.Errorf("%w: ada run berjalan di project", ErrValidation)
	}

	// Charge 1 kredit (auth_billing.md 3.4 langkah 1-4).
	chargeID, err := u.deps.Ledger.ChargeForRegenerate(ctx, art.UserID, art.ID, 1)
	if err != nil {
		if errors.Is(err, port.ErrInsufficientCredits) {
			return nil, ErrInsufficientCredits
		}
		return nil, wrap(err)
	}

	// Generate ulang tahap itu. Gagal = refund charge idempoten.
	regen := u.regenerator
	if regen == nil {
		u.refundRegenerate(ctx, art.UserID, chargeID)
		return nil, ErrUnavailable
	}
	content, err := regen.RegenerateStage(ctx, art)
	if err != nil {
		u.refundRegenerate(ctx, art.UserID, chargeID)
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// Simpan: versi lama ke artifact_versions, artifacts.content+version naik.
	if err := u.deps.Artifacts.InsertVersion(ctx, &entity.ArtifactVersion{
		ID: u.ids().New(), ArtifactID: art.ID, Version: art.Version,
		Content: art.Content, CreatedByRunID: &art.RunID, CreatedAt: u.clock(),
	}); err != nil {
		u.refundRegenerate(ctx, art.UserID, chargeID)
		return nil, wrap(err)
	}
	art.Version++
	art.Content = content
	art.Status = entity.ArtifactDraft
	art.UpdatedAt = u.clock()
	if err := u.deps.Artifacts.Update(ctx, art); err != nil {
		u.refundRegenerate(ctx, art.UserID, chargeID)
		return nil, wrap(err)
	}
	// Hilir stale (planner.md bagian 7).
	if err := u.deps.Artifacts.MarkDownstreamStale(ctx, art.RunID, art.Kind); err != nil {
		// Kegagalan ini tidak boleh membatalkan regenerate; dicatat log.
		if u.logger != nil {
			u.logger.Warn("tandai hilir stale gagal", "artifact", art.ID, "err", err)
		}
	}
	return art, nil
}

// refundRegenerate sekali (uq_ledger_one_refund_per_charge membuat replay no-op).
func (u *RunUsecase) refundRegenerate(ctx context.Context, userID entity.ID, chargeID entity.ID) {
	if err := u.deps.Ledger.RefundCharge(ctx, userID, chargeID, 1); err != nil && u.logger != nil {
		u.logger.Error("refund regenerate gagal", "charge", chargeID, "err", err)
	}
}
