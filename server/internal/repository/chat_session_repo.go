package repository

import (
	"context"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ChatSessionRepo akses chat_sessions (migrasi 0008).
//
// Satu proyek boleh punya banyak sesi percakapan. Daftar sesi diurutkan
// updated_at DESC supaya sesi yang baru dipakai muncul di atas.
type ChatSessionRepo struct{ db dbtx }

// NewChatSessionRepo merakit repository sesi chat.
func NewChatSessionRepo(db dbtx) *ChatSessionRepo { return &ChatSessionRepo{db: db} }

const chatSessionCols = `id, project_id, user_id, judul, created_at, updated_at`

func (r *ChatSessionRepo) Create(ctx context.Context, s *entity.ChatSession) error {
	_, err := r.db.Exec(ctx,
		`insert into chat_sessions (id, project_id, user_id, judul, created_at, updated_at)
		 values ($1,$2,$3,$4,$5,$6)`,
		s.ID, s.ProjectID, s.UserID, s.Judul, s.CreatedAt, s.UpdatedAt)
	return mapErr(err)
}

func (r *ChatSessionRepo) GetByID(ctx context.Context, id entity.ID) (*entity.ChatSession, error) {
	var s entity.ChatSession
	err := r.db.QueryRow(ctx,
		`select `+chatSessionCols+` from chat_sessions where id = $1`, id).
		Scan(&s.ID, &s.ProjectID, &s.UserID, &s.Judul, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

func (r *ChatSessionRepo) ListByProject(ctx context.Context, projectID entity.ID) ([]entity.ChatSession, error) {
	rows, err := r.db.Query(ctx,
		`select `+chatSessionCols+` from chat_sessions
		 where project_id = $1
		 order by updated_at desc, created_at desc`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var out []entity.ChatSession
	for rows.Next() {
		var s entity.ChatSession
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.UserID, &s.Judul,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, s)
	}
	return out, mapErr(rows.Err())
}

func (r *ChatSessionRepo) Delete(ctx context.Context, id entity.ID) error {
	_, err := r.db.Exec(ctx, `delete from chat_sessions where id = $1`, id)
	return mapErr(err)
}

func (r *ChatSessionRepo) Touch(ctx context.Context, id entity.ID, at time.Time) error {
	_, err := r.db.Exec(ctx, `update chat_sessions set updated_at = $2 where id = $1`, id, at)
	return mapErr(err)
}

var _ port.ChatSessionRepo = (*ChatSessionRepo)(nil)
