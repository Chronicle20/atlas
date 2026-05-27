package projection

import (
	"encoding/json"
	"sync"

	"atlas-login/configuration"
	"atlas-login/configuration/tenant"

	"github.com/google/uuid"
)

// State is the in-memory snapshot of service+tenant config that drives
// listener Add/Drain decisions. Concurrent reads are RW-locked; writes
// (ApplyService, ApplyTenant, …) are serialized by the subscriber's
// single goroutine.
type State struct {
	mu      sync.RWMutex
	service *configuration.RestModel // nil until first ApplyService
	tenants map[uuid.UUID]tenant.RestModel
}

func NewState() *State {
	return &State{tenants: make(map[uuid.UUID]tenant.RestModel)}
}

// ApplyService decodes the service config from env.Config and stores it.
// Returns an error on decode failure; the caller (subscriber) should log
// and skip without crashing.
func (s *State) ApplyService(env ServiceEnvelope) error {
	var cfg configuration.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
		return err
	}
	id, err := uuid.Parse(env.Id)
	if err != nil {
		return err
	}
	cfg.Id = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.service = &cfg
	return nil
}

// ApplyServiceTombstone clears the service config. After this call,
// Snapshot's first return is nil. atlas-login cannot run without a
// service config — the apply loop will respond by draining every
// listener if this transition is observed.
func (s *State) ApplyServiceTombstone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.service = nil
}

// ApplyTenant inserts or replaces the tenant config for env.Id.
func (s *State) ApplyTenant(env TenantEnvelope) error {
	var cfg tenant.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
		return err
	}
	id, err := uuid.Parse(env.Id)
	if err != nil {
		return err
	}
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

// Snapshot returns the current service config + a copy of the tenants
// map. The tenants map is copied so apply-loop iteration is decoupled
// from concurrent writes.
func (s *State) Snapshot() (*configuration.RestModel, map[uuid.UUID]tenant.RestModel) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var svc *configuration.RestModel
	if s.service != nil {
		c := *s.service
		svc = &c
	}
	out := make(map[uuid.UUID]tenant.RestModel, len(s.tenants))
	for k, v := range s.tenants {
		out[k] = v
	}
	return svc, out
}
