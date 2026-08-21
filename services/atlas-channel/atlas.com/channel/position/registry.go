// Package position holds the process-local, last-known (x, y) the channel
// last folded out of a character's movement path, or wrote on an
// inner-portal teleport (PRD FR-4.4). It exists in its own package,
// separate from movement and session, so that session can clear an entry on
// destroy without creating an import cycle: movement already imports
// session, so a registry living in movement could not be imported back by
// session. This package imports only sync and libs/atlas-tenant -- nothing
// from atlas-channel -- which is exactly what keeps session free to import
// it.
package position

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Key scopes a last-known position to one character in one tenant.
type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// Position is the authoritative (x, y) the channel last folded out of a
// movement path, or wrote on an inner-portal teleport.
type Position struct {
	X int16
	Y int16
}

// Registry is the process-wide last-known-position table. It carries no TTL
// and runs no sweeper: entries are bounded by the characters connected to
// this pod and are removed on session destroy, so unbounded growth is not a
// concern.
type Registry struct {
	mutex   sync.RWMutex
	entries map[Key]Position
}

var (
	registry *Registry
	once     sync.Once
)

// GetRegistry returns the process-wide last-known-position registry.
//
// In-process is the whole view, not a shard of it: a character's socket
// session lives on exactly one atlas-channel pod, so the last position that
// pod folded is the only one that matters.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{entries: make(map[Key]Position)}
	})
	return registry
}

// Put records the last-known position for a character.
func (r *Registry) Put(t tenant.Model, characterId uint32, p Position) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[Key{Tenant: t, CharacterId: characterId}] = p
}

// Lookup returns the last-known position for a character, if any.
func (r *Registry) Lookup(t tenant.Model, characterId uint32) (Position, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	p, ok := r.entries[Key{Tenant: t, CharacterId: characterId}]
	return p, ok
}

// Clear drops the last-known position for a character.
func (r *Registry) Clear(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.entries, Key{Tenant: t, CharacterId: characterId})
}
