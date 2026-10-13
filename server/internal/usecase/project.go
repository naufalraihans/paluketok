package usecase

import (
	"context"
	"errors"

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
	if err := u.deps.Projects.Create(ctx, p); err != nil {
		return nil, wrap(err)
	}
	return p, nil
}

// GetProject membaca project milik actor.
func (u *ProjectUsecase) GetProject(ctx context.Context, actor entity.Actor, id entity.ID) (*entity.Project, error) {
	p, err := u.ownedProject(ctx, actor, id)
	if err != nil {
		return nil, err
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
