// resolvedLLM menyambungkan port.LLM (tanpa Config) ke llm.Client dengan
// Config per slot. Sumber utama: baris ai_settings (diisi admin lewat panel,
// openapi.md bagian admin). Fallback env PLANNER_LLM_* untuk dev tanpa DB.
package main

import (
	"context"
	"errors"
	"os"

	"paluketok/server/internal/adapter/judge"
	"paluketok/server/internal/adapter/llm"
	"paluketok/server/internal/port"
)

// SlotLLM nama slot planner di ai_settings (erd.dbml ai_settings.slot).
const SlotLLM = "planner"

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

// resolvedJudge: JEV sudah pegang base+key di konstruktor; pass-through.
type resolvedJudge struct{ client *judge.Client }

func (r resolvedJudge) Decide(ctx context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	return r.client.Decide(ctx, req)
}
