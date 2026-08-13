package mist

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ProtectionRegistry is a tenant-scoped, in-memory index of live protection
// mists. Safe for concurrent use: the damage path reads it on every hit.
type ProtectionRegistry struct {
	mu        sync.RWMutex
	perTenant map[string]map[uuid.UUID]Protection
}

var (
	protectionRegistryOnce sync.Once
	protectionRegistry     *ProtectionRegistry
)

// GetProtectionRegistry returns the process-wide singleton, lazily built.
func GetProtectionRegistry() *ProtectionRegistry {
	protectionRegistryOnce.Do(func() {
		protectionRegistry = &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
	})
	return protectionRegistry
}

// NewTestProtectionRegistry returns a fresh, isolated registry so tests do
// not leak state through the singleton. Not used in production paths.
func NewTestProtectionRegistry() *ProtectionRegistry {
	return &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
}

// Add inserts p and lazily prunes entries that have already expired, so a
// dropped MIST_DESTROYED cannot accumulate stale rectangles.
func (r *ProtectionRegistry) Add(t tenant.Model, p Protection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		b = map[uuid.UUID]Protection{}
		r.perTenant[key] = b
	}
	now := time.Now()
	for id, existing := range b {
		if existing.Expired(now) {
			delete(b, id)
		}
	}
	b[p.Id()] = p
}

// Remove drops the protection with the given mist id. No-op if absent.
func (r *ProtectionRegistry) Remove(t tenant.Model, id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		return
	}
	delete(b, id)
	if len(b) == 0 {
		delete(r.perTenant, key)
	}
}

// Covering returns every live protection on f that contains (x, y). Expired
// entries are treated as absent regardless of whether they have been pruned:
// a missed MIST_DESTROYED must degrade to "no protection", never to a
// permanently invulnerable rectangle.
func (r *ProtectionRegistry) Covering(t tenant.Model, f field.Model, x, y int16, now time.Time) []Protection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[t.Id().String()]
	if !ok {
		return nil
	}
	var out []Protection
	for _, p := range b {
		if p.Expired(now) || !p.Field().Equals(f) || !p.Contains(x, y) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Len reports how many protections the tenant currently holds. Exposed for
// tests asserting the pruning behaviour.
func (r *ProtectionRegistry) Len(t tenant.Model) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.perTenant[t.Id().String()])
}
