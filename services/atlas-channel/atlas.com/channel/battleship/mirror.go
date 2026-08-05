package battleship

import (
	"sync"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RideState is the pod-local truth that a character is currently riding the
// battleship. SkillLevel is the caster's trained 5221006 level (from the
// buff event) so the break and lazy re-init paths need no skill-book fetch.
// StateTTL is the effect-derived TTL used to refresh the Redis ship-HP
// entry on every drain (FR-5.2 idle-expiry); 0 means "use the fallback".
type RideState struct {
	SkillLevel byte
	StateTTL   time.Duration
}

// RideMirror is a per-channel-process, in-memory projection of battleship
// MONSTER_RIDING buff events, read by the damage and attack hot paths with
// zero I/O. A character's socket session lives on exactly one channel pod,
// so the pod that receives its damage/attack packets is the pod whose buff
// consumer populated this mirror. Pattern precedent: monster.StatusMirror.
type RideMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]RideState
}

var (
	rideMirrorOnce sync.Once
	rideMirror     *RideMirror
)

// GetRideMirror returns the process-wide singleton mirror.
func GetRideMirror() *RideMirror {
	rideMirrorOnce.Do(func() {
		rideMirror = &RideMirror{perTenant: map[uuid.UUID]map[uint32]RideState{}}
	})
	return rideMirror
}

// Put records that characterId is currently riding the battleship, keyed
// per-tenant so a lookup under one tenant can never observe another
// tenant's rider.
func (m *RideMirror) Put(t tenant.Model, characterId uint32, s RideState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		tenantMap = map[uint32]RideState{}
		m.perTenant[t.Id()] = tenantMap
	}
	tenantMap[characterId] = s
}

// Get returns the ride state for characterId, if the mirror has one.
func (m *RideMirror) Get(t tenant.Model, characterId uint32) (RideState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return RideState{}, false
	}
	s, ok := tenantMap[characterId]
	return s, ok
}

// Remove clears characterId's ride state, if any. Idempotent.
func (m *RideMirror) Remove(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tenantMap, ok := m.perTenant[t.Id()]; ok {
		delete(tenantMap, characterId)
	}
}

// EvictTenant drops every entry for the tenant (listener drain).
func (m *RideMirror) EvictTenant(tid uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.perTenant, tid)
}
