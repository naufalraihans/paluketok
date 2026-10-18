// Repo ai_settings (erd.dbml 158-170): slot unik, api_key plaintext di DB
// (keputusan docs/plan/openapi.md - GET mengirim masked lewat usecase).
package repository

import (
	"context"
	"errors"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

type AISettingsRepo struct{ db dbtx }

const aiSettingCols = `slot, provider, base_url, model, api_key,
	max_input_tokens, max_output_tokens, is_active, updated_by`

func scanAISetting(row scannable) (*port.AISetting, error) {
	var s port.AISetting
	var updatedBy *entity.ID
	err := row.Scan(&s.Slot, &s.Provider, &s.BaseURL, &s.Model, &s.APIKey,
		&s.MaxInputTokens, &s.MaxOutputTokens, &s.IsActive, &updatedBy)
	if err != nil {
		return nil, mapErr(err)
	}
	s.UpdatedBy = updatedBy
	return &s, nil
}

func (r *AISettingsRepo) List(ctx context.Context) ([]port.AISetting, error) {
	rows, err := r.db.Query(ctx, `select `+aiSettingCols+` from ai_settings order by slot`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []port.AISetting
	for rows.Next() {
		s, err := scanAISetting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, mapErr(rows.Err())
}

func (r *AISettingsRepo) Get(ctx context.Context, slot string) (*port.AISetting, error) {
	return scanAISetting(r.db.QueryRow(ctx,
		`select `+aiSettingCols+` from ai_settings where slot=$1`, slot))
}

// Upsert menyeluruh set kolom per slot; slot baru dibuat bila belum ada.
func (r *AISettingsRepo) Upsert(ctx context.Context, s *port.AISetting, updatedBy entity.ID) error {
	tag, err := r.db.Exec(ctx,
		`insert into ai_settings (slot, provider, base_url, model, api_key,
		   max_input_tokens, max_output_tokens, is_active, updated_by, updated_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
		 on conflict (slot) do update set provider=$2, base_url=$3, model=$4,
		   api_key=$5, max_input_tokens=$6, max_output_tokens=$7, is_active=$8,
		   updated_by=$9, updated_at=now()`,
		s.Slot, s.Provider, s.BaseURL, s.Model, s.APIKey,
		s.MaxInputTokens, s.MaxOutputTokens, s.IsActive, updatedBy)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return mapErr(err)
		}
		return mapErr(err)
	}
	_ = tag
	return nil
}
