package configuration

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fetcher resolves a tenant's trade configuration. The default fetcher hits
// atlas-tenants; tests inject a stub so the cache logic can be exercised
// without a live HTTP call.
type fetcher func(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error)

// Registry is a lazy, per-tenant config cache. A fetch miss or error falls back
// to DefaultConfig so the service never hard-fails because a tenant has not
// configured trading (FR-9.2) — and never silently disables it either. Because
// the outcome is cached, the fallback notice is logged once per tenant.
type Registry struct {
	mu    sync.RWMutex
	cache map[uuid.UUID]Model
	fetch fetcher
}

var (
	registryOnce sync.Once
	registry     *Registry
)

// GetRegistry returns the process-wide trade-config registry singleton.
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
// it into the domain Model. A tier table that fails FR-9.3 validation is
// reported LOUDLY here; Extract then substitutes the shipped table so the
// service keeps running on known-good rates rather than on a broken one.
func defaultFetcher(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (Model, error) {
	rm, err := requestForTenant(tenantId)(l, ctx)
	if err != nil {
		return Model{}, err
	}

	tiers := make([]Tier, 0, len(rm.TaxTiers))
	for _, t := range rm.TaxTiers {
		tiers = append(tiers, Tier{Threshold: t.Threshold, Rate: t.Rate})
	}
	if len(tiers) > 0 {
		if verr := ValidateTiers(tiers); verr != nil {
			l.WithError(verr).Errorf("Tenant [%s] configured an invalid trade tax tier table. Falling back to the shipped default tiers.", tenantId.String())
		}
	}

	return Extract(rm), nil
}

// Get returns the cached configuration for the request's tenant, fetching and
// caching it on first access. On a missing tenant, a fetch miss or a fetch
// error it returns DefaultConfig — resolving configuration must never be the
// thing that fails a trade.
func (r *Registry) Get(l logrus.FieldLogger, ctx context.Context) Model {
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		l.WithError(err).Warn("No tenant in context when resolving trade configuration. Using defaults.")
		return DefaultConfig()
	}
	return r.getForTenant(l, ctx, t.Id())
}

// getForTenant is Get's cache body, split out so the tenant resolution above
// stays readable. Uses a read-locked fast path with a double-checked write lock.
func (r *Registry) getForTenant(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) Model {
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
		l.WithError(err).Infof("Tenant [%s] has no trade-configs resource. Using the shipped trade defaults.", tenantId.String())
		cfg = DefaultConfig()
	}
	r.cache[tenantId] = cfg
	return cfg
}
