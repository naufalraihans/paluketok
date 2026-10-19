// Store in-memory untuk E2E langkah 29 (workflow.md). Satu memStore
// menyimpan data; adapter tipis memenuhi tiap port (signature GetByID
// Project/Run/Artifact/Order berbeda, jadi tak bisa satu tipe).
package httpapi

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// LedgerEntryRef baris ledger ringkas untuk cek saldo/acuan idempoten.
type LedgerEntryRef struct {
	User    entity.ID
	Delta   int
	Reason  string
	RefType string
	RefID   entity.ID
	Key     string
}

type memStore struct {
	mu        sync.Mutex
	profiles  map[entity.ID]*entity.Profile
	projects  map[entity.ID]*entity.Project
	runs      map[entity.ID]*entity.PlanRun
	artifacts map[entity.ID]*entity.Artifact
	versions  []entity.ArtifactVersion
	decisions []entity.Decision
	orders    map[entity.ID]*entity.Order
	ledger    []LedgerEntryRef
	messages  []entity.ProjectMessage

	// sessions: sesi chat per proyek (migrasi 0008). Satu proyek boleh banyak.
	sessions []entity.ChatSession

	// skills: instruksi perancangan tersimpan (migrasi 0009). Global.
	skills []entity.Skill

	// planStages: skema tahap dinamis per run (migrasi 0007).
	planStages []entity.PlanStage
}

func newMemStore() *memStore {
	return &memStore{
		profiles:  map[entity.ID]*entity.Profile{},
		projects:  map[entity.ID]*entity.Project{},
		runs:      map[entity.ID]*entity.PlanRun{},
		artifacts: map[entity.ID]*entity.Artifact{},
		orders:    map[entity.ID]*entity.Order{},
	}
}

// memMessages adaptor port.MessageRepo di atas memStore.
type memMessages struct{ s *memStore }

func (m memMessages) Create(_ context.Context, msg *entity.ProjectMessage) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	cp := *msg
	m.s.messages = append(m.s.messages, cp)
	return nil
}

// ListByProject: percakapan satu proyek, urut waktu, dipotong `limit` pesan
// TERAKHIR (sama seperti versi SQL: yang dikirim ke model adalah ekor riwayat).
func (m memMessages) ListByProject(_ context.Context, projectID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var own []entity.ProjectMessage
	for _, msg := range m.s.messages {
		if msg.ProjectID == projectID {
			own = append(own, msg)
		}
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].CreatedAt.Before(own[j].CreatedAt) })
	if limit > 0 && len(own) > limit {
		own = own[len(own)-limit:]
	}
	return own, nil
}

// ListBySession: pesan SATU sesi saja. Inilah yang membedakannya dari
// ListByProject: riwayat antar sesi tidak boleh bercampur, karena itu sumber bug
// "kamu yakin?" yang dijawab dengan mengulang jawaban topik lain.
func (m memMessages) ListBySession(_ context.Context, sessionID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var own []entity.ProjectMessage
	for _, msg := range m.s.messages {
		if msg.SessionID == sessionID {
			own = append(own, msg)
		}
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].CreatedAt.Before(own[j].CreatedAt) })
	if limit > 0 && len(own) > limit {
		own = own[len(own)-limit:]
	}
	return own, nil
}

// memSessions fake chat_sessions. Daftar diurutkan updated_at DESC seperti SQL.
type memSessions struct{ s *memStore }

func (m memSessions) Create(_ context.Context, sess *entity.ChatSession) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	m.s.sessions = append(m.s.sessions, *sess)
	return nil
}

func (m memSessions) GetByID(_ context.Context, id entity.ID) (*entity.ChatSession, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i := range m.s.sessions {
		if m.s.sessions[i].ID == id {
			cp := m.s.sessions[i]
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (m memSessions) ListByProject(_ context.Context, projectID entity.ID) ([]entity.ChatSession, error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	var own []entity.ChatSession
	for _, sess := range m.s.sessions {
		if sess.ProjectID == projectID {
			own = append(own, sess)
		}
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].UpdatedAt.After(own[j].UpdatedAt) })
	return own, nil
}

func (m memSessions) Delete(_ context.Context, id entity.ID) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	kept := m.s.sessions[:0]
	for _, sess := range m.s.sessions {
		if sess.ID != id {
			kept = append(kept, sess)
		}
	}
	m.s.sessions = kept
	// Cascade seperti DB: pesan milik sesi ini ikut hilang.
	var keptMsg []entity.ProjectMessage
	for _, msg := range m.s.messages {
		if msg.SessionID != id {
			keptMsg = append(keptMsg, msg)
		}
	}
	m.s.messages = keptMsg
	return nil
}

func (m memSessions) Touch(_ context.Context, id entity.ID, at time.Time) error {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	for i := range m.s.sessions {
		if m.s.sessions[i].ID == id {
			m.s.sessions[i].UpdatedAt = at
		}
	}
	return nil
}

var _ port.ChatSessionRepo = memSessions{}

func pageSlice[T any](items []T, p port.Page) []T {
	if p.Limit > 0 {
		if p.Offset >= len(items) {
			return nil
		}
		end := p.Offset + p.Limit
		if end > len(items) {
			end = len(items)
		}
		return items[p.Offset:end]
	}
	return items
}

// seedProfile menaruh profil (kredit awal).
func (s *memStore) seedProfile(id entity.ID, credits int) *entity.Profile {
	p := &entity.Profile{ID: id, Credits: credits, PlanTier: entity.TierFree}
	s.profiles[id] = p
	return p
}

func nowUTC() time.Time { return time.Now().UTC() }

// memSkills fake port.SkillRepo untuk E2E endpoint admin skill.
//
// Nama unik ditegakkan seperti kolom `unique` di DB: tanpa itu, tes tidak akan
// pernah melihat jalur 409 dan bug-nya lolos ke prod.
type memSkills struct{ st *memStore }

func (m memSkills) Create(_ context.Context, s *entity.Skill) error {
	for _, x := range m.st.skills {
		if x.Nama == s.Nama {
			// Tiru error pgx apa adanya: repo memetakannya lewat pgconn.PgError
			// (kode 23505). Mengembalikan errors.New biasa akan lolos dari
			// pemetaan dan menyembunyikan jalur 409 dari tes.
			return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"skills_nama_key\""}
		}
	}
	m.st.skills = append(m.st.skills, *s)
	return nil
}

func (m memSkills) List(_ context.Context) ([]entity.Skill, error) {
	out := append([]entity.Skill(nil), m.st.skills...)
	sort.Slice(out, func(i, j int) bool { return out[i].Nama < out[j].Nama })
	return out, nil
}

func (m memSkills) GetByName(_ context.Context, nama string) (*entity.Skill, error) {
	for i := range m.st.skills {
		if m.st.skills[i].Nama == nama {
			cp := m.st.skills[i]
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (m memSkills) Update(_ context.Context, s *entity.Skill) error {
	for i := range m.st.skills {
		if m.st.skills[i].ID == s.ID {
			m.st.skills[i].Ringkasan = s.Ringkasan
			m.st.skills[i].Isi = s.Isi
			m.st.skills[i].UpdatedAt = s.UpdatedAt
			return nil
		}
	}
	return port.ErrNotFound
}

func (m memSkills) Delete(_ context.Context, id entity.ID) error {
	for i := range m.st.skills {
		if m.st.skills[i].ID == id {
			m.st.skills = append(m.st.skills[:i], m.st.skills[i+1:]...)
			return nil
		}
	}
	return port.ErrNotFound
}
