// AdminUsecase: endpoint admin (openapi.md bagian admin, workflow.md 26).
// Semua method menuntut actor.IsAdmin; sumber kebenaran profiles.is_admin
// dijalankan middleware JWT -> Actor (F5). Tanpa jalur tulis is_admin.
package usecase

import (
	"context"
	"errors"
	"os"
	"time"

	"paluketok/server/internal/adapter/llm"
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
	// model, provider, dan batas token TIDAK diminta dari admin: kolomnya
	// NOT NULL tanpa default, dan admin tidak mungkin tahu batas token tiap
	// penyedia. Isi dari slot lama bila ada, sisanya dari nilaiDefaultSlot.
	if old, err := u.settings.Get(ctx, s.Slot); err == nil && old != nil {
		if s.Provider == "" {
			s.Provider = old.Provider
		}
		if s.MaxInputTokens == 0 {
			s.MaxInputTokens = old.MaxInputTokens
		}
		if s.MaxOutputTokens == 0 {
			s.MaxOutputTokens = old.MaxOutputTokens
		}
	}
	nilaiDefaultSlot(s)
	// base_url harus http(s) dan host ada di allowlist (security.md bagian 12/14).
	if err := entity.ValidBaseURLAllowlist(s.BaseURL); err != nil {
		return ErrValidation
	}
	if err := u.settings.Upsert(ctx, s, actor.UserID); err != nil {
		return wrap(err)
	}
	return nil
}

// nilaiDefaultSlot mengisi kolom yang tidak diisi admin, HANYA bila slot ini
// belum pernah ada di database. Admin cukup mengisi base_url, model, dan api_key.
//
// Batas token diambil dari penyedia masing-masing: judge JEV memakai
// 8000/2000 (nilai yang sudah dipakai di produksi), planner 32000/8000.
// Angka ini cuma nilai awal, bukan batas atas: admin tetap boleh menimpanya
// lewat API bila penyedianya sanggup lebih.
func nilaiDefaultSlot(s *port.AISetting) {
	if s.Provider == "" {
		s.Provider = s.Slot
	}
	if s.MaxInputTokens <= 0 {
		s.MaxInputTokens = 8000
	}
	if s.MaxOutputTokens <= 0 {
		s.MaxOutputTokens = 2000
	}
	if s.Slot == "planner" {
		if s.MaxInputTokens == 8000 {
			s.MaxInputTokens = 32000
		}
		if s.MaxOutputTokens == 2000 {
			s.MaxOutputTokens = 8000
		}
	}
}

// TestAISetting memanggil penyedia dengan kredensial yang ada di form, TANPA
// menyimpan apa pun. api_key kosong = pakai key tersimpan di slot itu, supaya
// admin bisa menguji tanpa mengetik ulang key yang sudah tersimpan.
//
// Hasil pertama selalu dicoba sebagai model dari form; kalau modelnya kosong,
// dipakai model tersimpan.
func (u *AdminUsecase) TestAISetting(ctx context.Context, actor entity.Actor, in port.AISetting) (bool, time.Duration, error) {
	if err := adminOnly(actor); err != nil {
		return false, 0, err
	}
	if in.BaseURL == "" || in.Model == "" {
		return false, 0, ErrValidation
	}
	if err := entity.ValidBaseURLAllowlist(in.BaseURL); err != nil {
		return false, 0, ErrValidation
	}
	key := in.APIKey
	if key == "" && u.settings != nil {
		if old, err := u.settings.Get(ctx, in.Slot); err == nil && old != nil {
			key = old.APIKey
		}
	}
	// Key dari env dipakai sebagai cadangan terakhir supaya tombol ini tetap
	// berguna saat slot belum pernah disimpan (mis. pasang baru).
	if key == "" {
		key = os.Getenv("JEV_API_KEY")
		if key == "" {
			key = os.Getenv("PLANNER_LLM_API_KEY")
		}
	}
	if key == "" {
		return false, 0, ErrValidation
	}
	start := time.Now()
	// Panggilan sengaja sangat kecil: yang diuji koneksi dan izin model, bukan
	// mutu jawaban. max_tokens kecil menekan biaya saat admin klik berkali-kali.
	_, err := llm.New().Complete(ctx, llm.Config{BaseURL: in.BaseURL, APIKey: key, Model: in.Model},
		port.LLMRequest{
			Messages:  []port.LLMMessage{{Role: "user", Content: "ping"}},
			MaxTokens: 1,
		})
	latency := time.Since(start)
	if err != nil {
		if llm.IsUnavailable(err) {
			return false, latency, nil
		}
		return false, latency, wrap(err)
	}
	return true, latency, nil
}

// maskKey: tampil 4 karakter pertama + "..." + 4 terakhir (contoh openapi.md).
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
