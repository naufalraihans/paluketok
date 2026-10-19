package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Batas isi skill. 20000 selaras dengan CHECK di migrasi 0009; dijaga juga di
// sini supaya pengguna dapat pesan yang bisa dibaca, bukan pelanggaran
// constraint mentah dari Postgres.
const (
	SkillMaxIsi       = 20000
	SkillMaxRingkasan = 200
)

// Nama skill dipanggil dengan garis miring di chat, jadi hanya huruf kecil,
// angka, dan tanda hubung. Huruf besar dan spasi akan memutus pemisahan
// perintah, dan nama panjang tanpa batas membuka jalan pesan raksasa.
var namaSkillPola = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// ListSkills daftar skill untuk panel admin (GET /v1/admin/skills).
func (u *AdminUsecase) ListSkills(ctx context.Context, actor entity.Actor) ([]entity.Skill, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	if u.skills == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	return u.skills.List(ctx)
}

// CreateSkill menambah skill (POST /v1/admin/skills).
func (u *AdminUsecase) CreateSkill(ctx context.Context, actor entity.Actor, nama, ringkasan, isi string) (*entity.Skill, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	if u.skills == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	nama = strings.ToLower(strings.TrimSpace(nama))
	ringkasan = strings.TrimSpace(ringkasan)
	if !namaSkillPola.MatchString(nama) {
		return nil, ErrValidation
	}
	if err := cekIsiSkill(ringkasan, isi); err != nil {
		return nil, err
	}
	now := u.clock.Now()
	s := &entity.Skill{
		ID:        u.ids.New(),
		Nama:      nama,
		Ringkasan: ringkasan,
		Isi:       isi,
		CreatedBy: &actor.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.skills.Create(ctx, s); err != nil {
		// Nama unik: tabrakan berarti sudah ada skill dengan nama itu. Lewat
		// wrap supaya port.ErrConflict diterjemahkan ke usecase.ErrConflict,
		// satu-satunya bentuk yang dipetakan handler ke 409.
		if isUniqueViolation(err) {
			return nil, wrap(port.ErrConflict)
		}
		return nil, wrap(err)
	}
	return s, nil
}

// UpdateSkill mengubah ringkasan dan isi. Nama TIDAK diubah: nama adalah cara
// skill dipanggil, dan menggantinya memutus percakapan yang menyebutnya.
func (u *AdminUsecase) UpdateSkill(ctx context.Context, actor entity.Actor, id, ringkasan, isi string) (*entity.Skill, error) {
	if err := adminOnly(actor); err != nil {
		return nil, err
	}
	if u.skills == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	ringkasan = strings.TrimSpace(ringkasan)
	if err := cekIsiSkill(ringkasan, isi); err != nil {
		return nil, err
	}
	s := &entity.Skill{
		ID:        entity.ID(id),
		Ringkasan: ringkasan,
		Isi:       isi,
		UpdatedAt: u.clock.Now(),
	}
	if err := u.skills.Update(ctx, s); err != nil {
		return nil, wrap(err)
	}
	// Dibaca ulang supaya pemanggil menerima nama dan waktu yang sebenarnya,
	// bukan objek setengah terisi.
	list, err := u.skills.List(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	for i := range list {
		if list[i].ID == s.ID {
			return &list[i], nil
		}
	}
	return nil, port.ErrNotFound
}

// DeleteSkill menghapus skill (DELETE /v1/admin/skills/{id}).
func (u *AdminUsecase) DeleteSkill(ctx context.Context, actor entity.Actor, id string) error {
	if err := adminOnly(actor); err != nil {
		return err
	}
	if u.skills == nil {
		return wrap(port.ErrUnavailable)
	}
	return wrap(u.skills.Delete(ctx, entity.ID(id)))
}

func cekIsiSkill(ringkasan, isi string) error {
	if utf8.RuneCountInString(ringkasan) > SkillMaxRingkasan {
		return ErrValidation
	}
	n := utf8.RuneCountInString(isi)
	if n == 0 || n > SkillMaxIsi {
		return ErrValidation
	}
	return nil
}

// isUniqueViolation true bila err pelanggaran unique (SQLSTATE 23505).
//
// Memakai errors.As ke *pgconn.PgError, BUKAN pencarian teks "23505" di
// err.Error(): pgconn.PgError.Error() hanya mengembalikan Message, sehingga
// kode SQLSTATE tidak ada di teksnya dan pencocokan teks gagal diam-diam.
// Akibatnya tabrakan nama lolos jadi 500 alih-alih 409.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
