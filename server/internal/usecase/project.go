package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ProjectUsecase menangani CRUD project. Semua operasi owner-scoped:
// plan/roles.md (user: CRUD milik sendiri) dan plan/architecture.md bagian 3.
type ProjectUsecase struct {
	deps Deps
}

func NewProjectUsecase(d Deps) *ProjectUsecase {
	if d.Clock == nil {
		panic("usecase: Clock wajib diisi")
	}
	if d.IDs == nil {
		panic("usecase: IDGenerator wajib diisi")
	}
	return &ProjectUsecase{deps: d}
}

// CreateProjectInput adalah input pembuatan project
// (plan/openapi.md POST /v1/projects: name, brief, stack).
type CreateProjectInput struct {
	Name  string
	Brief string
	Stack entity.Stack
}

// CreateProject membuat project milik actor. Actor datang dari JWT claim,
// bukan dari body (plan/roles.md "Aturan").
func (u *ProjectUsecase) CreateProject(ctx context.Context, actor entity.Actor, in CreateProjectInput) (*entity.Project, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	name, err := entity.NormalizeName(in.Name)
	if err != nil {
		return nil, asValidation(err)
	}
	if err := entity.ValidateBrief(in.Brief); err != nil {
		return nil, asValidation(err)
	}

	now := u.deps.Clock.Now()
	p := &entity.Project{
		ID:        u.deps.IDs.New(),
		OwnerID:   actor.UserID,
		Name:      name,
		Brief:     in.Brief,
		Stack:     in.Stack,
		Status:    entity.ProjectDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// URL proyek memakai slug (/projects/testing-jembut), bukan UUID.
	slug, err := u.uniqueSlug(ctx, actor.UserID, name)
	if err != nil {
		return nil, err
	}
	p.Slug = slug
	if err := u.deps.Projects.Create(ctx, p); err != nil {
		return nil, wrap(err)
	}
	return p, nil
}

// uniqueSlug menurunkan slug dari nama lalu memastikan belum dipakai owner ini
// dengan menambahkan akhiran -2, -3, ... Dibatasi 20 percobaan; lebih dari itu
// berarti ada yang tidak beres (dikembalikan sebagai error, bukan dipaksa).
func (u *ProjectUsecase) uniqueSlug(ctx context.Context, ownerID entity.ID, name string) (string, error) {
	base := entity.Slugify(name)
	if base == "" {
		// Nama hanya berisi tanda baca (mis. "***"): tetap butuh URL yang sah.
		base = "proyek"
	}
	for i := 1; i <= 20; i++ {
		candidate := base
		// Percobaan pertama memakai slug polos, KECUALI namanya kebetulan
		// berbentuk UUID: URL seperti itu akan ditangkap jalur UUID di
		// resolveProject dan berakhir 404, jadi langsung diberi akhiran.
		if i > 1 || entity.ValidID(base) {
			candidate = slugWithSuffix(base, i)
		}
		taken, err := u.deps.Projects.SlugTaken(ctx, ownerID, candidate)
		if err != nil {
			return "", wrap(err)
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", ErrValidation
}

// slugWithSuffix menambahkan "-n" ke slug dasar, memotong dasarnya bila total
// melewati batas panjang. Hasilnya selalu berakhir dengan angka, jadi tidak
// mungkin berbentuk UUID.
func slugWithSuffix(base string, n int) string {
	suffix := "-" + strconv.Itoa(n)
	if len(base)+len(suffix) > entity.SlugMaxLen {
		base = base[:entity.SlugMaxLen-len(suffix)]
	}
	return strings.Trim(base, "-") + suffix
}

// GetProject membaca project milik actor. ref boleh UUID (bentuk lama) ATAU
// slug (bentuk ramah baca) — tautan lama tetap hidup setelah URL berpindah.
func (u *ProjectUsecase) GetProject(ctx context.Context, actor entity.Actor, ref string) (*entity.Project, error) {
	p, err := u.resolveProject(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Resolve menerjemahkan ref (UUID atau slug) menjadi proyek MILIK actor dan
// menegakkan policy owner: bukan pemilik -> 403, tidak ada -> 404.
//
// Dipakai usecase lain (chat konteks proyek) supaya aturan 403-vs-404 dan
// penerjemahan slug hanya hidup di satu tempat.
func (u *ProjectUsecase) Resolve(ctx context.Context, actor entity.Actor, ref string) (*entity.Project, error) {
	return u.resolveProject(ctx, actor, ref)
}

// resolveProject menerima UUID atau slug dan mengembalikan project milik actor.
// Urutan pencarian: UUID -> slug. Proyek lama tanpa slug sudah di-backfill oleh
// migrasi 0005, jadi setiap proyek punya URL enak dibaca.
func (u *ProjectUsecase) resolveProject(ctx context.Context, actor entity.Actor, ref string) (*entity.Project, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	if entity.ValidID(ref) {
		return u.ownedProject(ctx, actor, entity.ID(ref))
	}
	if !entity.ValidSlug(ref) {
		return nil, ErrNotFound
	}
	// Slug unik per owner, jadi cari MILIK AKTOR dulu: kalau dua user punya
	// slug sama, tiap orang mendapat proyeknya sendiri (bukan 403 salah orang).
	p, err := u.deps.Projects.GetBySlug(ctx, actor.UserID, ref)
	if errors.Is(err, port.ErrNotFound) && !actor.IsAdmin {
		// Bukan miliknya. Cari tanpa filter owner: kalau ada milik user lain,
		// balas 403 (plan/security.md: 403, bukan 404); kalau tidak ada sama
		// sekali, 404.
		if _, gerr := u.deps.Projects.GetBySlug(ctx, "", ref); gerr == nil {
			return nil, ErrForbidden
		}
		return nil, ErrNotFound
	}
	if errors.Is(err, port.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, wrap(err)
	}
	if p.OwnerID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return p, nil
}

// ListProjects membaca daftar project milik actor.
func (u *ProjectUsecase) ListProjects(ctx context.Context, actor entity.Actor, page port.Page) ([]entity.Project, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	list, err := u.deps.Projects.ListByOwner(ctx, actor.UserID, pageOrDefault(page))
	if err != nil {
		return nil, wrap(err)
	}
	return list, nil
}

// UpdateProjectInput adalah input PATCH /v1/projects/{id}: brief, stack, status.
// Field nil berarti tidak diubah.
type UpdateProjectInput struct {
	Name   *string
	Brief  *string
	Stack  *entity.Stack
	Status *entity.ProjectStatus
}

// UpdateProject mengubah project milik actor.
func (u *ProjectUsecase) UpdateProject(ctx context.Context, actor entity.Actor, id entity.ID, in UpdateProjectInput) (*entity.Project, error) {
	p, err := u.ownedProject(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name, verr := entity.NormalizeName(*in.Name)
		if verr != nil {
			return nil, asValidation(verr)
		}
		p.Name = name
	}
	if in.Brief != nil {
		if verr := entity.ValidateBrief(*in.Brief); verr != nil {
			return nil, asValidation(verr)
		}
		p.Brief = *in.Brief
	}
	if in.Stack != nil {
		p.Stack = *in.Stack
	}
	if in.Status != nil {
		if !entity.ValidProjectStatus(*in.Status) {
			return nil, ErrValidation
		}
		p.Status = *in.Status
	}
	p.UpdatedAt = u.deps.Clock.Now()

	if err := u.deps.Projects.Update(ctx, p); err != nil {
		return nil, wrap(err)
	}
	return p, nil
}

// ArchiveProject menandai project archived. Bukan hapus baris
// (plan/erd.md "Aturan status", plan/openapi.md DELETE /v1/projects/{id}).
func (u *ProjectUsecase) ArchiveProject(ctx context.Context, actor entity.Actor, id entity.ID) error {
	if _, err := u.ownedProject(ctx, actor, id); err != nil {
		return err
	}
	return wrap(u.deps.Projects.Archive(ctx, id, actor.UserID, u.deps.Clock.Now()))
}

// ownedProject mengambil project lalu memastikan actornya pemilik atau admin.
// Admin boleh membaca semua (plan/roles.md matriks izin).
func (u *ProjectUsecase) ownedProject(ctx context.Context, actor entity.Actor, id entity.ID) (*entity.Project, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	if !entity.ValidID(string(id)) {
		return nil, ErrNotFound
	}
	p, err := u.deps.Projects.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	if p.OwnerID != actor.UserID && !actor.IsAdmin {
		return nil, ErrForbidden
	}
	return p, nil
}
