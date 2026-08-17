package configuration

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// defaultFetcher fetches a tenant's configuration from atlas-tenants and folds
// it into the domain Model.
func defaultFetcher(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error) {
	rm, err := requestForTenant(ctx, tenantId)(l, ctx)
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
		// A genuine 404 means the tenant simply hasn't provisioned
		// kite-configs yet -- expected and routine, so it's logged at Info.
		// Anything else (connection refused, 5xx, decode failure) is a real
		// atlas-tenants problem masquerading as tenant-defaulting, so it's
		// logged at Warn to stay visible. The fallback behaviour itself is
		// unchanged either way: an un-provisioned tenant must still work on
		// compiled defaults.
		if errors.Is(err, requests.ErrNotFound) {
			l.Infof("Tenant %s has not provisioned kite-configs, using defaults", tenantId.String())
		} else {
			l.WithError(err).Warnf("Failed to fetch kite config for tenant %s, using defaults", tenantId.String())
		}
		cfg = DefaultConfig()
	}
	r.cache[tenantId] = cfg
	return cfg
}
