package server

import (
	"sync"
)

var registry *Registry
var once sync.Once

// Registry holds the per-(tenant, world, channel) Model entries this
// process knows about. Backing map keyed by server.Key so Deregister
// and lookup are O(1); GetAll preserves the historical slice contract
// for callers that iterate every entry.
type Registry struct {
	lock    sync.RWMutex
	entries map[Key]Model
}

// GetRegistry returns the process-wide singleton. Exported so listener
// drains and projection apply loops can call Deregister without going
// through the free-function facade.
func GetRegistry() *Registry { return getRegistry() }

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{entries: make(map[Key]Model)}
	})
	return registry
}

// Register inserts or replaces the entry for KeyOf(m). Replacement on
// duplicate insert is intentional — a re-register after a partial drain
// should overwrite stale state, not stack.
func (r *Registry) Register(m Model) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.entries[KeyOf(m)] = m
}

// Deregister removes the entry at k. No-op if absent.
func (r *Registry) Deregister(k Key) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.entries, k)
}

// Get returns (model, true) when present, (zero, false) otherwise.
func (r *Registry) Get(k Key) (Model, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	m, ok := r.entries[k]
	return m, ok
}

// GetAll returns a snapshot copy. The slice is safe to mutate without
// affecting the registry; iteration order is undefined (map iteration).
func (r *Registry) GetAll() []Model {
	r.lock.RLock()
	defer r.lock.RUnlock()
	out := make([]Model, 0, len(r.entries))
	for _, m := range r.entries {
		out = append(out, m)
	}
	return out
}
