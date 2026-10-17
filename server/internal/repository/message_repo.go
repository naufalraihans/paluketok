package repository

import (
	"context"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// MessageRepo akses project_messages (chat konteks proyek).
type MessageRepo struct{ db dbtx }

// NewMessageRepo merakit repository pesan.
func NewMessageRepo(db dbtx) *MessageRepo { return &MessageRepo{db: db} }

const messageCols = `id, project_id, user_id, role, content, suggested_stage,
	model, tokens_in, tokens_out, created_at`

func (r *MessageRepo) Create(ctx context.Context, m *entity.ProjectMessage) error {
	_, err := r.db.Exec(ctx,
		`insert into project_messages
		 (id, project_id, user_id, role, content, suggested_stage, model, tokens_in, tokens_out, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.ID, m.ProjectID, m.UserID, m.Role, m.Content, nullIfEmpty(m.SuggestedStage),
		nullIfEmpty(m.Model), m.TokensIn, m.TokensOut, m.CreatedAt)
	return mapErr(err)
}

// ListByProject membaca percakapan urut lama -> baru.
//
// Dibatasi `limit` pesan TERAKHIR: percakapan panjang tidak boleh mengirim
// seluruh riwayat ke model. Subquery dipakai supaya yang dipotong adalah ekor
// riwayat, lalu diurutkan naik untuk dibaca.
func (r *MessageRepo) ListByProject(ctx context.Context, projectID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		`select `+messageCols+` from (
		   select `+messageCols+` from project_messages
		   where project_id = $1
		   order by created_at desc, id desc
		   limit $2
		 ) t order by created_at asc, id asc`,
		projectID, limit)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var out []entity.ProjectMessage
	for rows.Next() {
		var m entity.ProjectMessage
		var stage, model *string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.Content,
			&stage, &model, &m.TokensIn, &m.TokensOut, &m.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		if stage != nil {
			m.SuggestedStage = *stage
		}
		if model != nil {
			m.Model = *model
		}
		out = append(out, m)
	}
	return out, mapErr(rows.Err())
}

var _ port.MessageRepo = (*MessageRepo)(nil)
