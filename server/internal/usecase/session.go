package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Batas judul sesi. Kolom DB membatasi 1..120 karakter; dipotong di sini
// supaya judul panjang dari pesan pertama tidak menabrak CHECK dan gagal
// simpan hanya karena itu.
const (
	SessionJudulMax = 120
	// SesiMaxPerProject mencegah daftar sesi tumbuh tanpa batas lewat klik
	// berulang. Bukan batas teknis, batas kewarasan.
	SesiMaxPerProject = 50
)

// Sessions mengembalikan daftar sesi satu proyek, terbaru dulu.
func (u *ChatUsecase) Sessions(ctx context.Context, actor entity.Actor, ref string) ([]entity.ChatSession, error) {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	if u.deps.Sessions == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	list, err := u.deps.Sessions.ListByProject(ctx, p.ID)
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// NewSession membuat sesi kosong. Judul opsional: kosong = "Sesi baru".
func (u *ChatUsecase) NewSession(ctx context.Context, actor entity.Actor, ref, judul string) (*entity.ChatSession, error) {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	if u.deps.Sessions == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	existing, err := u.deps.Sessions.ListByProject(ctx, p.ID)
	if err != nil {
		return nil, wrap(err)
	}
	if len(existing) >= SesiMaxPerProject {
		return nil, ErrValidation
	}
	now := u.deps.Clock.Now()
	s := &entity.ChatSession{
		ID:        u.deps.IDs.New(),
		ProjectID: p.ID,
		UserID:    actor.UserID,
		Judul:     judulSesi(judul, ""),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.deps.Sessions.Create(ctx, s); err != nil {
		return nil, wrap(err)
	}
	return s, nil
}

// DeleteSession menghapus satu sesi beserta pesannya (cascade di DB).
//
// Sesi terakhir TIDAK boleh dihapus: proyek tanpa sesi membuat UI kosong dan
// pengguna harus menebak. Kalau ingin bersih, buat sesi baru dulu.
func (u *ChatUsecase) DeleteSession(ctx context.Context, actor entity.Actor, ref, sessionRef string) error {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return err
	}
	if u.deps.Sessions == nil {
		return wrap(port.ErrUnavailable)
	}
	s, err := u.sessionOwned(ctx, p, actor, sessionRef)
	if err != nil {
		return err
	}
	all, err := u.deps.Sessions.ListByProject(ctx, p.ID)
	if err != nil {
		return wrap(err)
	}
	if len(all) <= 1 {
		return ErrValidation
	}
	return wrap(u.deps.Sessions.Delete(ctx, s.ID))
}

// MessagesInSession mengembalikan pesan satu sesi, urut lama -> baru.
func (u *ChatUsecase) MessagesInSession(ctx context.Context, actor entity.Actor, ref, sessionRef string, page port.Page) ([]entity.ProjectMessage, error) {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	s, err := u.sessionOwned(ctx, p, actor, sessionRef)
	if err != nil {
		return nil, err
	}
	need := page.Limit + page.Offset
	ms, err := u.deps.Messages.ListBySession(ctx, s.ID, need)
	if err != nil {
		return nil, wrap(err)
	}
	if page.Offset > 0 {
		if page.Offset >= len(ms) {
			return nil, nil
		}
		ms = ms[page.Offset:]
	}
	return ms, nil
}

// sessionFor memilih sesi tujuan sebuah pesan.
//
// Alur: sessionRef yang diberikan -> sesi itu (wajib milik proyek ini);
// sessionRef kosong -> sesi terakhir proyek; proyek tanpa sesi -> buat baru
// dengan judul dari pesan pertama, karena itulah konteks yang tersedia.
func (u *ChatUsecase) sessionFor(ctx context.Context, p *entity.Project, actor entity.Actor, sessionRef, firstText string) (*entity.ChatSession, error) {
	if strings.TrimSpace(sessionRef) != "" {
		return u.sessionOwned(ctx, p, actor, sessionRef)
	}
	list, err := u.deps.Sessions.ListByProject(ctx, p.ID)
	if err != nil {
		return nil, wrap(err)
	}
	if len(list) > 0 {
		return &list[0], nil
	}
	now := u.deps.Clock.Now()
	s := &entity.ChatSession{
		ID:        u.deps.IDs.New(),
		ProjectID: p.ID,
		UserID:    actor.UserID,
		Judul:     judulSesi("", firstText),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.deps.Sessions.Create(ctx, s); err != nil {
		return nil, wrap(err)
	}
	return s, nil
}

// sessionOwned mengambil sesi dan memastikan ia milik proyek yang diminta.
//
// Pemeriksaan proyek penting: tanpa itu, id sesi proyek lain bisa dipakai untuk
// membaca percakapan orang lain lewat proyek milik sendiri.
func (u *ChatUsecase) sessionOwned(ctx context.Context, p *entity.Project, actor entity.Actor, sessionRef string) (*entity.ChatSession, error) {
	if u.deps.Sessions == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	s, err := u.deps.Sessions.GetByID(ctx, entity.ID(strings.TrimSpace(sessionRef)))
	if err != nil {
		if err == port.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	if s.ProjectID != p.ID {
		return nil, ErrNotFound
	}
	if s.UserID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return s, nil
}

// judulSesi menentukan judul sesi: judul eksplisit bila ada, kalau tidak
// potongan pesan pertama, kalau tidak juga "Sesi baru".
func judulSesi(explicit, dariPesan string) string {
	s := strings.TrimSpace(explicit)
	if s == "" {
		s = strings.TrimSpace(dariPesan)
	}
	if s == "" {
		return "Sesi baru"
	}
	// Ambil baris pertama saja: pesan panjang tidak layak jadi judul.
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return clip(berhentiDiKata(s, SessionJudulMax), SessionJudulMax)
}

// berhentiDiKata memotong pada batas kata terakhir sebelum max rune, supaya
// judul tidak berakhir separuh kata. Kalau tidak ada spasi, potong apa adanya.
func berhentiDiKata(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	head := string(r[:max])
	if i := strings.LastIndexAny(head, " \t"); i > 0 {
		return head[:i]
	}
	return head
}
