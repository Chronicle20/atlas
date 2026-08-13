package buff

import (
	"sync"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// EnergyMirror tracks each character's ENERGY_CHARGE bar reading from buff
// APPLIED / STAT_UPDATED / EXPIRED events, so the Energy Blast cast gate can
// read it with zero I/O on the attack hot path (task-216 design.md §4.4).
//
// Values are the raw stat amounts: 0..10000 while accumulating, and the 15000
// sentinel while charged. A missing entry means "unknown", NOT "empty bar" —
// callers must treat the two differently, because this mirror is
// process-local and repopulates only from subsequent events after a channel
// restart or a channel change.
type EnergyMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]int32
}

var (
	energyMirror     *EnergyMirror
	energyMirrorOnce sync.Once
)

// GetEnergyMirror returns the process-wide singleton mirror, lazily
// initialising it on first call.
func GetEnergyMirror() *EnergyMirror {
	energyMirrorOnce.Do(func() {
		energyMirror = &EnergyMirror{perTenant: make(map[uuid.UUID]map[uint32]int32)}
	})
	return energyMirror
}

// Set records or replaces the tenant/character's bar reading.
func (m *EnergyMirror) Set(t tenant.Model, characterId uint32, value int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		c = make(map[uint32]int32)
		m.perTenant[t.Id()] = c
	}
	c[characterId] = value
}

// Clear removes the tenant/character's bar reading, returning the gate to its
// fail-open "unknown" state.
func (m *EnergyMirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.perTenant[t.Id()]; ok {
		delete(c, characterId)
	}
}

// Get returns the tenant/character's bar reading, if one is known.
func (m *EnergyMirror) Get(t tenant.Model, characterId uint32) (int32, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		return 0, false
	}
	v, ok := c[characterId]
	return v, ok
}
