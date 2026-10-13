// Adapter port di atas memStore (E2E langkah 29). Nama method bebas karena
// tiap adapter memenuhi tepat satu port.
package httpapi

import (
	"context"
	"sort"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ---------- ProfileRepo ----------

type memProfiles struct{ s *memStore }

func (m memProfiles) GetByID(_ context.Context, id entity.ID) (*entity.Profile, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.profiles[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m memProfiles) SetPlanTier(_ context.Context, id entity.ID, tier entity.PlanTier) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.profiles[id]
	if !ok {
		return port.ErrNotFound
	}
	p.PlanTier = tier
	return nil
}

func (m memProfiles) IsAdmin(_ context.Context, id entity.ID) (bool, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.profiles[id]
	if !ok {
		return false, port.ErrNotFound
	}
	return p.IsAdmin, nil
}

// ---------- ProjectRepo ----------

type memProjects struct{ s *memStore }

func (m memProjects) Create(_ context.Context, p *entity.Project) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	cp := *p
	m.s.projects[p.ID] = &cp
	return nil
}

func (m memProjects) GetByID(_ context.Context, id entity.ID) (*entity.Project, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.projects[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m memProjects) ListByOwner(_ context.Context, owner entity.ID, page port.Page) ([]entity.Project, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.Project
	for _, p := range m.s.projects {
		if p.OwnerID == owner {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return pageSlice(out, page), nil
}

func (m memProjects) ListAll(_ context.Context, page port.Page) ([]entity.Project, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.Project
	for _, p := range m.s.projects {
		out = append(out, *p)
	}
	return pageSlice(out, page), nil
}

func (m memProjects) Update(_ context.Context, p *entity.Project) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if _, ok := m.s.projects[p.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *p
	m.s.projects[p.ID] = &cp
	return nil
}

func (m memProjects) Archive(_ context.Context, id, owner entity.ID, _ time.Time) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	p, ok := m.s.projects[id]
	if !ok || p.OwnerID != owner {
		return port.ErrNotFound
	}
	p.Status = entity.ProjectArchived
	return nil
}
