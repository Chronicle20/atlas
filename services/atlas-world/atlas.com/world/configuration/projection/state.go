package projection

import (
	"atlas-world/configuration/tenant"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// State is the in-memory snapshot of tenant config. Concurrent reads are
// RW-locked; writes are serialized by the subscriber's single goroutine.
type State struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]tenant.RestModel
}

func NewState() *State {
	return &State{tenants: make(map[uuid.UUID]tenant.RestModel)}
}

// ApplyTenant inserts or replaces the tenant config for env.Id. The
// tenant.RestModel.Id field is json:"-" (absent from the envelope config
// payload), so it is populated explicitly from env.Id to keep the
// snapshot model identical to the previously REST-loaded one.
func (s *State) ApplyTenant(env TenantEnvelope) error {
	var cfg tenant.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
		return err
	}
	id, err := uuid.Parse(env.Id)
	if err != nil {
		return err
	}
	cfg.Id = env.Id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[id] = cfg
	return nil
}

// ApplyTenantTombstone removes the tenant config for id.
func (s *State) ApplyTenantTombstone(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, id)
}

// Snapshot returns a copy of the tenants map so callers iterate decoupled
// from concurrent writes.
func (s *State) Snapshot() map[uuid.UUID]tenant.RestModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(s.tenants))
	for k, v := range s.tenants {
		out[k] = v
	}
	return out
}
