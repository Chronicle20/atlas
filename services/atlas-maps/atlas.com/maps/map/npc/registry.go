package npc

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// FieldKey scopes a registry entry to a tenant and field instance.
//
// NPC placements are session/instance-scoped — Cosmic's
// AbstractPlayerInteraction.spawnNpc is not persisted, and is expected to be
// re-created (by the onUserEnter guard, or a script) on the next field entry
// rather than surviving a restart. That matches map/weather's in-memory
// Registry, not map/monster's Redis-backed SpawnPointRegistry, which persists
// spawn-point cooldowns across restarts on purpose (task-290 G2, C14).
type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

// Registry holds the scripted NPCs currently placed on each field, keyed by
// tenant and field. A field may hold more than one NPC (including duplicates
// of the same npcId), so entries are stored as a slice.
type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]Model
	nextId  uint32
}

var (
	registry *Registry
	once     sync.Once
)

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{entries: make(map[FieldKey][]Model)}
	})
	return registry
}

// NextId issues a monotonically increasing unique ID for a newly created NPC.
func (r *Registry) NextId() uint32 {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.nextId++
	return r.nextId
}

// Add records a newly created NPC against its field key.
func (r *Registry) Add(key FieldKey, m Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[key] = append(r.entries[key], m)
}

// GetAll returns a copy of the NPCs currently placed on the given field key.
func (r *Registry) GetAll(key FieldKey) []Model {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	existing := r.entries[key]
	out := make([]Model, len(existing))
	copy(out, existing)
	return out
}

// Reset clears every registered NPC, across every tenant and field. Primarily
// used for testing.
func (r *Registry) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries = make(map[FieldKey][]Model)
}
