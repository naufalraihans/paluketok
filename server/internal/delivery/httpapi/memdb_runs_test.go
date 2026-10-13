// Adapter RunRepo + RunStore di atas memStore (E2E langkah 29).
package httpapi

import (
	"context"
	"sort"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

type memRuns struct{ s *memStore }

func (m memRuns) GetByID(_ context.Context, id entity.ID) (*entity.PlanRun, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	r, ok := m.s.runs[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (m memRuns) ListByProject(_ context.Context, pid entity.ID) ([]entity.PlanRun, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.PlanRun
	for _, r := range m.s.runs {
		if r.ProjectID == pid {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (m memRuns) ListAllRuns(_ context.Context, status string, uid entity.ID, page port.Page) ([]entity.PlanRun, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []entity.PlanRun
	for _, r := range m.s.runs {
		if status != "" && string(r.Status) != status {
			continue
		}
		if uid != "" && r.UserID != uid {
			continue
		}
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return pageSlice(out, page), nil
}

func (m memRuns) HasRunning(_ context.Context, pid entity.ID) (bool, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, r := range m.s.runs {
		if r.ProjectID == pid && r.Status == entity.RunRunning {
			return true, nil
		}
	}
	return false, nil
}

func (m memRuns) HasOtherRunning(_ context.Context, pid, exclude entity.ID) (bool, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for _, r := range m.s.runs {
		if r.ProjectID == pid && r.ID != exclude && r.Status == entity.RunRunning {
			return true, nil
		}
	}
	return false, nil
}

func (m memRuns) Update(_ context.Context, r *entity.PlanRun) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if _, ok := m.s.runs[r.ID]; !ok {
		return port.ErrNotFound
	}
	cp := *r
	m.s.runs[r.ID] = &cp
	return nil
}

// --- RunStore (worker) ---

type memRunStore struct{ s *memStore }

func (m memRunStore) GetFull(ctx context.Context, id entity.ID) (*entity.PlanRun, error) {
	return memRuns(m).GetByID(ctx, id)
}

func (m memRunStore) ClaimQueued(_ context.Context) (*port.QueuedRun, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var oldest *entity.PlanRun
	for _, r := range m.s.runs {
		if r.Status == entity.RunQueued && (oldest == nil || r.CreatedAt.Before(oldest.CreatedAt)) {
			cp := *r
			oldest = &cp
		}
	}
	if oldest == nil {
		return nil, port.ErrNotFound
	}
	oldest.Status = entity.RunRunning
	now := nowUTC()
	oldest.StartedAt = &now
	return &port.QueuedRun{ID: oldest.ID, ProjectID: oldest.ProjectID, UserID: oldest.UserID}, nil
}

func (m memRunStore) SaveProgress(_ context.Context, id entity.ID, stage entity.Stage, progress int, status entity.RunStatus) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	r, ok := m.s.runs[id]
	if !ok {
		return port.ErrNotFound
	}
	st := stage
	r.CurrentStage = &st
	r.Progress = progress
	r.Status = status
	return nil
}

func (m memRunStore) Finish(_ context.Context, id entity.ID, st entity.RunStatus, errMsg string, now time.Time) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	r, ok := m.s.runs[id]
	if !ok {
		return port.ErrNotFound
	}
	r.Status = st
	if errMsg != "" {
		e := errMsg
		r.Error = &e
	}
	if st != entity.RunRunning {
		r.FinishedAt = &now
	}
	return nil
}

func (m memRunStore) HasOtherRunning(_ context.Context, pid, exclude entity.ID) (bool, error) {
	return memRuns(m).HasOtherRunning(context.Background(), pid, exclude)
}

func (m memRunStore) ListQueued(_ context.Context) ([]port.QueuedRun, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var out []port.QueuedRun
	for _, r := range m.s.runs {
		if r.Status == entity.RunQueued {
			out = append(out, port.QueuedRun{ID: r.ID, ProjectID: r.ProjectID, UserID: r.UserID})
		}
	}
	return out, nil
}

func (m memRunStore) RequeueRunning(_ context.Context, now time.Time) (int, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	n := 0
	for _, r := range m.s.runs {
		if r.Status == entity.RunRunning {
			r.Status = entity.RunFailed
			e := "server_restart"
			r.Error = &e
			r.FinishedAt = &now
			n++
		}
	}
	return n, nil
}
