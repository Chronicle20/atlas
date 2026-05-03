package timer

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type tenantBucket struct {
	tenant  tenant.Model
	entries map[uint32]Entry
}

type Registry struct {
	mu        sync.RWMutex
	perTenant map[string]*tenantBucket
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[string]*tenantBucket{}}
	})
	return registry
}

func NewTestRegistry() *Registry {
	return &Registry{perTenant: map[string]*tenantBucket{}}
}

func tenantKey(t tenant.Model) string {
	return t.Id().String()
}

func (r *Registry) bucket(t tenant.Model) *tenantBucket {
	key := tenantKey(t)
	b, ok := r.perTenant[key]
	if !ok {
		b = &tenantBucket{tenant: t, entries: map[uint32]Entry{}}
		r.perTenant[key] = b
	}
	return b
}

// Add inserts or replaces the entry for (e.Tenant(), e.CharacterId()). Replacement
// is silent — callers that care about pre-existing entries should call Cancel
// first to obtain the prior entry's stop handle.
func (r *Registry) Add(e Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.bucket(e.Tenant())
	b.entries[e.CharacterId()] = e
	return nil
}

func (r *Registry) Get(t tenant.Model, characterId uint32) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	return e, ok
}

// Cancel atomically removes and returns the entry. The caller is responsible
// for stopping the entry's underlying time.Timer.
func (r *Registry) Cancel(t tenant.Model, characterId uint32) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	if !ok {
		return Entry{}, false
	}
	delete(b.entries, characterId)
	if len(b.entries) == 0 {
		delete(r.perTenant, tenantKey(t))
	}
	return e, true
}
