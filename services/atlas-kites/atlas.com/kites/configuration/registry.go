package configuration

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// fetcher resolves a tenant's kite configuration. The default fetcher hits
// atlas-tenants; tests inject a stub so the cache logic can be exercised
// without a live HTTP call.
type fetcher func(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error)

// Registry is a lazy, per-tenant config cache. A fetch miss or error falls back
// to DefaultConfig so the service never hard-fails because a tenant has not yet
// configured kite placement policy.
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

// SetFetcherForTest overrides the process-wide singleton's fetch function,
// bypassing the atlas-tenants HTTP round trip entirely. Exported for
// cross-package tests -- atlas-kites/kite's processor tests need config
// resolution without a live HTTP server, and its per-map-cap concurrency
// test needs to force a specific policy knob (maxPerMap: 1) -- rather than
// inventing a second config-loading path. Returns a restore function; the
// existing per-tenant cache is untouched, which is safe because every test
// resolves a fresh tenant.Create(uuid.New(), ...) so no two tests share a
// cache entry.
func SetFetcherForTest(f func(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error)) func() {
	r := GetRegistry()
	r.mu.Lock()
	old := r.fetch
	r.fetch = f
	r.mu.Unlock()
	return func() {
		r.mu.Lock()
		r.fetch = old
		r.mu.Unlock()
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
		l.WithError(err).Warnf("Failed to fetch kite config for tenant %s, using defaults", tenantId.String())
		cfg = DefaultConfig()
	}
	r.cache[tenantId] = cfg
	return cfg
}
