package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// MessageRepo akses project_messages (chat konteks proyek).
type MessageRepo struct{ db dbtx }

// NewMessageRepo merakit repository pesan.
func NewMessageRepo(db dbtx) *MessageRepo { return &MessageRepo{db: db} }

const messageCols = `id, project_id, user_id, session_id, role, content, suggested_stage,
	model, tokens_in, tokens_out, created_at`

func (r *MessageRepo) Create(ctx context.Context, m *entity.ProjectMessage) error {
	_, err := r.db.Exec(ctx,
		`insert into project_messages
		 (id, project_id, user_id, session_id, role, content, suggested_stage, model, tokens_in, tokens_out, created_at)
		 values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		m.ID, m.ProjectID, m.UserID, m.SessionID, m.Role, m.Content,
		nullIfEmpty(m.SuggestedStage), nullIfEmpty(m.Model), m.TokensIn, m.TokensOut, m.CreatedAt)
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
	return scanMessages(rows, err)
}

// ListBySession membaca satu sesi saja, urut lama -> baru.
//
// Ini yang dipakai chat: riwayat satu proyek bercampur antar sesi, sehingga
// pertanyaan pendek seperti "kamu yakin?" dijawab dengan mengulang jawaban
// topik lain. Dibatasi ekor riwayat dengan pola yang sama seperti ListByProject.
func (r *MessageRepo) ListBySession(ctx context.Context, sessionID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		`select `+messageCols+` from (
		   select `+messageCols+` from project_messages
		   where session_id = $1
		   order by created_at desc, id desc
		   limit $2
		 ) t order by created_at asc, id asc`,
		sessionID, limit)
	return scanMessages(rows, err)
}

// scanMessages menyeragamkan pemetaan baris pesan untuk kedua pembaca.
func scanMessages(rows pgx.Rows, err error) ([]entity.ProjectMessage, error) {
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var out []entity.ProjectMessage
	for rows.Next() {
		var m entity.ProjectMessage
		var session, stage, model *string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &session, &m.Role, &m.Content,
			&stage, &model, &m.TokensIn, &m.TokensOut, &m.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		if session != nil {
			m.SessionID = entity.ID(*session)
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
