// AdminUsecase: endpoint admin (openapi.md bagian admin, workflow.md 26).
// Semua method menuntut actor.IsAdmin; sumber kebenaran profiles.is_admin
// dijalankan middleware JWT -> Actor (F5). Tanpa jalur tulis is_admin.
package usecase

import (
	"context"
	"errors"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// AdminSettingsRepo port ai_settings (boleh nil di test yang tak menyentuhnya).
type AdminSettingsRepo = port.AISettingsRepo

// AdminUsecase baca lintas user untuk dashboard admin.
type AdminUsecase struct {
	projects  port.ProjectRepo
	runs      port.RunRepo
	decisions port.DecisionRepo
	profiles  port.ProfileRepo
	ledger    port.LedgerRepo
	settings  AdminSettingsRepo
}

func NewAdminUsecase(d Deps, settings AdminSettingsRepo) *AdminUsecase {
	return &AdminUsecase{projects: d.Projects, runs: d.Runs, decisions: d.Decisions,
		profiles: d.Profiles, ledger: d.Ledger, settings: settings}
}

func adminOnly(actor entity.Actor) error {
	if actor.UserID == "" || !actor.IsAdmin {
		return ErrForbidden
	}
	return nil
}

// ListProjects semua project (GET /v1/admin/projects).
func (u *AdminUsecase) ListProjects(ctx context.Context, actor entity.Actor, page port.Page) ([]entity.Project, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	list, err := u.projects.ListAll(ctx, page)
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// ListRuns semua run, filter status/user_id (GET /v1/admin/runs).
func (u *AdminUsecase) ListRuns(ctx context.Context, actor entity.Actor, status string, userID entity.ID, page port.Page) ([]entity.PlanRun, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	list, err := u.runs.ListAllRuns(ctx, status, userID, page)
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// ListDecisions log JEV semua user (GET /v1/admin/decisions).
// state & criteria mentah TIDAK ikut respons (aturan openapi.md).
func (u *AdminUsecase) ListDecisions(ctx context.Context, actor entity.Actor, page port.Page) ([]entity.Decision, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	list, err := u.decisions.ListAll(ctx, page)
	if err != nil {
		return nil, wrap(err)
	}
	for i := range list {
		list[i].State = nil
		list[i].Criteria = nil
	}
	return list, nil
}

// SetPlanTier (PATCH /v1/admin/users/{id}). is_admin tak tersentuh.
func (u *AdminUsecase) SetPlanTier(ctx context.Context, actor entity.Actor, userID entity.ID, tier entity.PlanTier) error {
	if err := adminOnly(actor); err != nil {
		return err
	}
	if !entity.ValidID(string(userID)) {
		return ErrNotFound
	}
	if !entity.ValidPlanTier(tier) {
		return ErrValidation
	}
	if err := u.profiles.SetPlanTier(ctx, userID, tier); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return wrap(err)
	}
	return nil
}

// ListAISettings daftar slot AI, api_key masked (GET /v1/admin/settings/ai).
func (u *AdminUsecase) ListAISettings(ctx context.Context, actor entity.Actor) ([]port.AISetting, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	list, err := u.settings.List(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	for i := range list {
		list[i].APIKey = maskKey(list[i].APIKey)
	}
	return list, nil
}

// SetAISetting (PUT /v1/admin/settings/ai/{slot}).
func (u *AdminUsecase) SetAISetting(ctx context.Context, actor entity.Actor, s *port.AISetting) error {
	if err := adminOnly(actor); err != nil {
		return err
	}
	if s.Slot == "" || s.BaseURL == "" || s.Model == "" {
		return ErrValidation
	}
	// base_url harus http(s) dan host ada di allowlist (security.md bagian 12/14).
	if err := entity.ValidBaseURLAllowlist(s.BaseURL); err != nil {
		return ErrValidation
	}
	if err := u.settings.Upsert(ctx, s, actor.UserID); err != nil {
		return wrap(err)
	}
	return nil
}

// maskKey: tampil 4 karakter pertama + "..." + 4 terakhir (contoh openapi.md).
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
