package mist

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// ErrNotFound is returned when a mist with the given id is not present in the
// registry for the given tenant.
var ErrNotFound = errors.New("mist not found")

// ErrAlreadyExists is returned when Add is called with a mist whose id is
// already present in the tenant's bucket.
var ErrAlreadyExists = errors.New("mist with id already exists")

// tenantBucket holds the full tenant.Model alongside the per-id mist map so
// callers iterating tenants do not have to round-trip through a tenant
// registry.
type tenantBucket struct {
	tenant tenant.Model
	mists  map[uuid.UUID]Mist
}

// Registry is a tenant-scoped, in-memory index of active Mist values. It is
// safe for concurrent use.
type Registry struct {
	mu        sync.RWMutex
	perTenant map[string]*tenantBucket
}

var (
	registryOnce sync.Once
	registry     *Registry
)

// GetRegistry returns the process-wide singleton Registry, lazily constructed
// on first call.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[string]*tenantBucket{}}
	})
	return registry
}

func tenantKey(t tenant.Model) string {
	return t.Id().String()
}

func (r *Registry) bucket(t tenant.Model) *tenantBucket {
	key := tenantKey(t)
	b, ok := r.perTenant[key]
	if !ok {
		b = &tenantBucket{tenant: t, mists: map[uuid.UUID]Mist{}}
		r.perTenant[key] = b
	}
	return b
}

// Add inserts m into the registry under the given tenant. ErrAlreadyExists is
// returned if a mist with the same id is already present in the tenant's
// bucket.
func (r *Registry) Add(t tenant.Model, m Mist) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.bucket(t)
	if _, exists := b.mists[m.Id()]; exists {
		return ErrAlreadyExists
	}
	b.mists[m.Id()] = m
	return nil
}

// Remove deletes the mist with the given id from the tenant's bucket and
// returns it. ErrNotFound is returned if no such mist exists.
func (r *Registry) Remove(t tenant.Model, id uuid.UUID) (Mist, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Mist{}, ErrNotFound
	}
	m, ok := b.mists[id]
	if !ok {
		return Mist{}, ErrNotFound
	}
	delete(b.mists, id)
	if len(b.mists) == 0 {
		delete(r.perTenant, tenantKey(t))
	}
	return m, nil
}

// GetByField returns all mists in the tenant's bucket whose field matches f
// (including instance UUID).
func (r *Registry) GetByField(t tenant.Model, f field.Model) []Mist {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return []Mist{}
	}
	out := make([]Mist, 0)
	for _, m := range b.mists {
		if m.Field().Equals(f) {
			out = append(out, m)
		}
	}
	return out
}

// AllByTenant returns every mist registered for the tenant across all fields.
func (r *Registry) AllByTenant(t tenant.Model) []Mist {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return []Mist{}
	}
	out := make([]Mist, 0, len(b.mists))
	for _, m := range b.mists {
		out = append(out, m)
	}
	return out
}

// UpdateLastTick advances the lastTick timestamp on the mist with the given
// id. No-op if the mist is not present.
func (r *Registry) UpdateLastTick(t tenant.Model, id uuid.UUID, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return
	}
	m, ok := b.mists[id]
	if !ok {
		return
	}
	b.mists[id] = m.WithLastTick(at)
}

// GetTenants returns a snapshot of every tenant.Model currently holding at
// least one mist.
func (r *Registry) GetTenants() []tenant.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]tenant.Model, 0, len(r.perTenant))
	for _, b := range r.perTenant {
		out = append(out, b.tenant)
	}
	return out
}
