// Store in-memory untuk E2E langkah 29 (workflow.md). Satu memStore
// menyimpan data; adapter tipis memenuhi tiap port (signature GetByID
// Project/Run/Artifact/Order berbeda, jadi tak bisa satu tipe).
package httpapi

import (
	"sync"
	"time"

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

// Page slicing default: limit<=0 = semua (pola repo asli memakai limit
// default dari pageFrom, di sini cukup untuk E2E).
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
