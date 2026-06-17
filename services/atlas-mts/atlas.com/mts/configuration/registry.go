package configuration

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// fetcher resolves a tenant's MTS configuration. The default fetcher hits
// atlas-tenants; tests inject a stub so the cache logic can be exercised
// without a live HTTP call.
type fetcher func(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error)

// Registry is a lazy, per-tenant config cache. A fetch miss or error falls back
// to DefaultConfig so the service never hard-fails because a tenant has not yet
// configured the MTS (the atlas-tenants resource lands in Phase 8).
type Registry struct {
	mu    sync.RWMutex
	cache map[uuid.UUID]Model
	fetch fetcher
}

var (
	registryOnce sync.Once
	registry     *Registry
)

// GetRegistry returns the process-wide config registry singleton.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = newRegistryWithFetcher(defaultFetcher)
	})
	return registry
}

// newRegistryWithFetcher constructs a registry with an explicit fetcher. The
// default fetcher is wired in GetRegistry; tests inject a stub here.
func newRegistryWithFetcher(f fetcher) *Registry {
	return &Registry{
		cache: make(map[uuid.UUID]Model),
		fetch: f,
	}
}

// defaultFetcher fetches a tenant's configuration from atlas-tenants and folds
// it into the domain Model.
func defaultFetcher(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error) {
	rm, err := requestForTenant(tenantId)(l, ctx)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm), nil
}

// GetTenantConfig returns the cached config for the request's tenant, fetching
// and caching it on first access. On a fetch miss or error it caches and
// returns DefaultConfig so subsequent calls stay cheap and the service degrades
// gracefully. Uses a read-locked fast path with a double-checked write lock.
func (r *Registry) GetTenantConfig(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) Model {
	r.mu.RLock()
	if cfg, ok := r.cache[tenantId]; ok {
		r.mu.RUnlock()
		return cfg
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if cfg, ok := r.cache[tenantId]; ok {
		return cfg
	}

	cfg, err := r.fetch(l, ctx, tenantId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch MTS config for tenant %s, using defaults", tenantId.String())
		cfg = DefaultConfig()
	}
	r.cache[tenantId] = cfg
	return cfg
}
