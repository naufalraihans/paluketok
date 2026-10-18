// Package repository mengimplementasikan port repository dengan pgx.
// Semua query lewat service role: filter owner/user manual ada di usecase
// lapis 2 dan di query list/owner (docs/plan/security.md bagian 3).
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// dbtx adalah kemampuan minimum pool dan tx supaya helper bisa dipakai
// di keduanya.
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// scannable dipakai helper scan bersama (pgx.Row dan pgx.Rows sama-sama punya).
type scannable interface {
	Scan(dest ...any) error
}

// Repos kumpulkan semua implementasi port dari satu pool.
type Repos struct {
	Profiles   *ProfileRepo
	Projects   *ProjectRepo
	Runs       *RunRepo
	Artifacts  *ArtifactRepo
	Decisions  *DecisionRepo
	Orders     *OrderRepo
	Ledger     *LedgerRepo
	AISettings *AISettingsRepo
	Messages   *MessageRepo
}

func New(pool *pgxpool.Pool) *Repos {
	return &Repos{
		Profiles:   &ProfileRepo{pool},
		Projects:   &ProjectRepo{pool},
		Runs:       &RunRepo{pool},
		Artifacts:  &ArtifactRepo{pool},
		Decisions:  &DecisionRepo{pool},
		Orders:     &OrderRepo{pool},
		Ledger:     &LedgerRepo{pool},
		AISettings: &AISettingsRepo{pool},
		Messages:   &MessageRepo{pool},
	}
}

// mapErr memetakan pgx.ErrNoRows ke port.ErrNotFound.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return port.ErrNotFound
	}
	return err
}

// --- ProfileRepo ---

type ProfileRepo struct{ db dbtx }

func (r *ProfileRepo) GetByID(ctx context.Context, id entity.ID) (*entity.Profile, error) {
	var p entity.Profile
	err := r.db.QueryRow(ctx,
		`select id, username, full_name, avatar_url, plan_tier, is_admin, credits, token_version, created_at
		 from profiles where id = $1`, id).
		Scan(&p.ID, &p.Username, &p.FullName, &p.AvatarURL, &p.PlanTier,
			&p.IsAdmin, &p.Credits, &p.TokenVersion, &p.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// IsAdmin membaca kolom is_admin satu user; dipakai JWT middleware (F7).
func (r *ProfileRepo) IsAdmin(ctx context.Context, id entity.ID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`select is_admin from profiles where id = $1`, id).Scan(&ok)
	if err != nil {
		return false, mapErr(err)
	}
	return ok, nil
}
