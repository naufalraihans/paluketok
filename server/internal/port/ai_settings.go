package port

import (
	"context"

	"paluketok/server/internal/entity"
)

// AISetting satu baris ai_settings (erd.dbml 158-170). APIKey plaintext di
// DB (keputusan plan), masked saat dikirim ke klien oleh usecase.
type AISetting struct {
	Slot            string
	Provider        string
	BaseURL         string
	Model           string
	APIKey          string
	MaxInputTokens  int
	MaxOutputTokens int
	IsActive        bool
	UpdatedBy       *entity.ID
}

// AISettingsRepo akses ai_settings (admin F7). Tanpa cache; pembacaan jarang.
type AISettingsRepo interface {
	List(ctx context.Context) ([]AISetting, error)
	Get(ctx context.Context, slot string) (*AISetting, error)
	Upsert(ctx context.Context, s *AISetting, updatedBy entity.ID) error
}
