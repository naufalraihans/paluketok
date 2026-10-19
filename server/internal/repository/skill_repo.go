package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// SkillRepo akses tabel `skills` (migrasi 0009).
//
// Cakupan GLOBAL: siapa saja boleh membaca, hanya admin yang menulis. Aturan itu
// ditegakkan di usecase DAN di RLS; repo ini tidak mengulanginya supaya pesan
// errornya datang dari satu tempat.
type SkillRepo struct{ db dbtx }

const skillCols = `id, nama, ringkasan, isi, created_by, created_at, updated_at`

func (r *SkillRepo) Create(ctx context.Context, s *entity.Skill) error {
	_, err := r.db.Exec(ctx, `
		insert into public.skills (id, nama, ringkasan, isi, created_by, created_at, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7)`,
		s.ID, s.Nama, s.Ringkasan, s.Isi, s.CreatedBy, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		// Nama unik: tabrakan dipetakan ke ErrConflict supaya handler menjawab
		// 409. Tanpa ini pengguna melihat 500 "kesalahan server" untuk kesalahan
		// yang bisa ia perbaiki sendiri (ganti nama).
		if isUnique(err) {
			return port.ErrConflict
		}
		// CHECK nama/isi (migrasi 0009) juga kesalahan pengguna, bukan server.
		if isCheck(err) {
			return port.ErrValidation
		}
		return err
	}
	return nil
}

func (r *SkillRepo) List(ctx context.Context) ([]entity.Skill, error) {
	rows, err := r.db.Query(ctx, `select `+skillCols+` from public.skills order by nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.Skill
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *SkillRepo) GetByName(ctx context.Context, nama string) (*entity.Skill, error) {
	rows, err := r.db.Query(ctx, `select `+skillCols+` from public.skills where nama = $1`, nama)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, port.ErrNotFound
	}
	return scanSkill(rows)
}

func (r *SkillRepo) Update(ctx context.Context, s *entity.Skill) error {
	tag, err := r.db.Exec(ctx, `
		update public.skills set ringkasan=$2, isi=$3, updated_at=$4 where id=$1`,
		s.ID, s.Ringkasan, s.Isi, s.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func (r *SkillRepo) Delete(ctx context.Context, id entity.ID) error {
	tag, err := r.db.Exec(ctx, `delete from public.skills where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return port.ErrNotFound
	}
	return nil
}

func scanSkill(rows pgx.Rows) (*entity.Skill, error) {
	var (
		s       entity.Skill
		created *time.Time
		updated *time.Time
	)
	if err := rows.Scan(&s.ID, &s.Nama, &s.Ringkasan, &s.Isi, &s.CreatedBy, &created, &updated); err != nil {
		return nil, err
	}
	if created != nil {
		s.CreatedAt = *created
	}
	if updated != nil {
		s.UpdatedAt = *updated
	}
	return &s, nil
}

var _ port.SkillRepo = (*SkillRepo)(nil)
