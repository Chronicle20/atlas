package buff

import (
	"sync"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// BeaconEntry is the channel-local projection of a character's active
// HOMING_BEACON lock (statup amount = locked monster object id).
type BeaconEntry struct {
	sourceId int32
	level    byte
	mobId    int32
}

// NewBeaconEntry constructs a BeaconEntry from its constituent fields.
func NewBeaconEntry(sourceId int32, level byte, mobId int32) BeaconEntry {
	return BeaconEntry{
		sourceId: sourceId,
		level:    level,
		mobId:    mobId,
	}
}

func (e BeaconEntry) SourceId() int32 {
	return e.sourceId
}

func (e BeaconEntry) Level() byte {
	return e.level
}

func (e BeaconEntry) MobId() int32 {
	return e.mobId
}

// BeaconMirror tracks each character's active beacon from buff APPLIED /
// EXPIRED events so the local-give path can re-carry the populated
// GuidedBullet block on unrelated gives (design.md §3 F2: pre-95 clients
// overwrite the stored beacon from every local give trailer).
//
// Process-local by design: after a channel restart it repopulates only from
// subsequent events, so an unrelated give to a still-locked character may
// drop the lock visual until re-cast (accepted degradation, design.md §5.7).
type BeaconMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]BeaconEntry
}

var (
	beaconMirror     *BeaconMirror
	beaconMirrorOnce sync.Once
)

// GetBeaconMirror returns the process-wide singleton mirror, lazily
// initialising it on first call.
func GetBeaconMirror() *BeaconMirror {
	beaconMirrorOnce.Do(func() {
		beaconMirror = &BeaconMirror{perTenant: make(map[uuid.UUID]map[uint32]BeaconEntry)}
	})
	return beaconMirror
}

// Set records or replaces the tenant/character's active beacon. Re-setting
// an already-locked character replaces the entry (re-casting the beacon on
// a different monster moves the lock).
func (m *BeaconMirror) Set(t tenant.Model, characterId uint32, e BeaconEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		c = make(map[uint32]BeaconEntry)
		m.perTenant[t.Id()] = c
	}
	c[characterId] = e
}

// Clear removes the tenant/character's active beacon, if any.
func (m *BeaconMirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.perTenant[t.Id()]; ok {
		delete(c, characterId)
	}
}

// Get returns the tenant/character's active beacon, if any.
func (m *BeaconMirror) Get(t tenant.Model, characterId uint32) (BeaconEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		return BeaconEntry{}, false
	}
	e, ok := c[characterId]
	return e, ok
}
