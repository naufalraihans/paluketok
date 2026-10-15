// resolvedLLM / resolvedJudge menyambungkan port ke adapter masing-masing
// dengan Config per slot dari ai_settings (diisi admin lewat panel,
// openapi.md bagian admin). Fallback env untuk dev tanpa DB.
//
// Dua slot dipakai (erd.dbml ai_settings.slot):
//   - planner -> LLM penyusun enam dokumen planning
//   - judge   -> gateway JEV (gate keputusan, plan/jev_ai.md)
//
// Keduanya dibaca per panggilan, jadi ganti key dari admin berlaku tanpa restart.
package main

import (
	"context"
	"errors"
	"os"

	"paluketok/server/internal/adapter/judge"
	"paluketok/server/internal/adapter/llm"
	"paluketok/server/internal/port"
)

// Slot planner + judge di ai_settings (erd.dbml ai_settings.slot).
const (
	SlotLLM   = "planner"
	SlotJudge = "judge"
)

type resolvedLLM struct {
	client *llm.Client
	store  port.AISettingsRepo // nil = pakai env saja
}

// config ambil slot dari DB; bila kosong/gagal, jatuh ke env.
func (r resolvedLLM) config(ctx context.Context) (llm.Config, error) {
	if r.store != nil {
		if s, err := r.store.Get(ctx, SlotLLM); err == nil && s.BaseURL != "" && s.Model != "" {
			return llm.Config{BaseURL: s.BaseURL, APIKey: s.APIKey, Model: s.Model}, nil
		}
	}
	cfg := llm.Config{
		BaseURL: os.Getenv("PLANNER_LLM_BASE_URL"),
		APIKey:  os.Getenv("PLANNER_LLM_API_KEY"),
		Model:   os.Getenv("PLANNER_LLM_MODEL"),
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return cfg, errors.New("slot " + SlotLLM + " belum diisi admin (settings/ai)")
	}
	return cfg, nil
}

func (r resolvedLLM) Complete(ctx context.Context, req port.LLMRequest) (*port.LLMResponse, error) {
	cfg, err := r.config(ctx)
	if err != nil {
		return nil, err
	}
	return r.client.Complete(ctx, cfg, req)
}

// resolvedJudge: base+key JEV diambil per panggilan dari slot ai_settings,
// fallback env JEV_BASE_URL/JEV_API_KEY (plan/jev_ai.md bagian 1).
// Model JEV tetap "jev-latest" (konstanta di adapter, bukan dari DB).
type resolvedJudge struct {
	client *judge.Client
	store  port.AISettingsRepo // nil = pakai env saja
}

func (r resolvedJudge) Decide(ctx context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	base, key := "", ""
	if r.store != nil {
		if s, err := r.store.Get(ctx, SlotJudge); err == nil && s.BaseURL != "" {
			base, key = s.BaseURL, s.APIKey
		}
	}
	if base == "" {
		base, key = os.Getenv("JEV_BASE_URL"), os.Getenv("JEV_API_KEY")
	}
	if key == "" {
		return nil, errors.New("slot " + SlotJudge + " belum diisi admin (settings/ai)")
	}
	// Client tipis per panggilan: http.Client di dalamnya stateless
	// (transport punya connection pool sendiri), jadi tidak perlu cache.
	c := judge.New(base, key)
	return c.Decide(ctx, req)
}
