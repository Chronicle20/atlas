package skill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// In-process TTL cache for immutable atlas-data skill templates (task-122
// FR-4.3). Semantics ported from task-060 (atlas-monsters
// monster/information/cache.go) re-expressed in memory per task-120 §5.4:
// positive/negative TTLs, ErrNotFound-only negative caching, env
// kill-switch, lazy expiry, no singleflight.

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

const (
	envEnabled     = "SKILL_DATA_CACHE_ENABLED"
	envTTL         = "SKILL_DATA_CACHE_TTL"
	envNegativeTTL = "SKILL_DATA_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute

	outcomeCacheHit         = "hit"
	outcomeCacheNegativeHit = "negative_hit"
	outcomeCacheMiss        = "miss"
)

// configLogger is the logger used for one-time configuration warnings.
var configLogger logrus.FieldLogger = logrus.StandardLogger()

func loadConfig() cacheConfig {
	return cacheConfig{
		enabled:     parseBoolEnv(envEnabled, true),
		ttl:         parseDurationEnv(envTTL, defaultTTL, minTTL, maxTTL),
		negativeTTL: parseDurationEnv(envNegativeTTL, defaultNegativeTTL, minNegativeTTL, maxNegativeTTL),
	}
}

func parseBoolEnv(name string, def bool) bool {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	switch v {
	case "true", "TRUE", "True", "1", "yes", "y":
		return true
	case "false", "FALSE", "False", "0", "no", "n":
		return false
	default:
		configLogger.Warnf("invalid bool for %s=%q; using default %v", name, v, def)
		return def
	}
}

func parseDurationEnv(name string, def, lo, hi time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		configLogger.Warnf("invalid duration for %s=%q; using default %s", name, v, def)
		return def
	}
	if d < lo || d > hi {
		configLogger.Warnf("%s=%s out of range [%s, %s]; using default %s", name, d, lo, hi, def)
		return def
	}
	return d
}

type cacheEntry struct {
	m         Model
	negative  bool
	expiresAt time.Time
}

type skillCache struct {
	cfg       cacheConfig
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]cacheEntry
}

var (
	skillCacheOnce sync.Once
	skillCachePtr  *skillCache
)

func getSkillCache() *skillCache {
	skillCacheOnce.Do(func() {
		skillCachePtr = &skillCache{
			cfg:       loadConfig(),
			perTenant: map[uuid.UUID]map[uint32]cacheEntry{},
		}
	})
	return skillCachePtr
}

// upstreamFn is the test seam for the real REST fetch (task-060 precedent).
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, skillId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(ctx, skillId), Extract)()
}

// notFoundError synthesizes a not-found error for negative-cache hits so
// callers see the same errors.Is(err, requests.ErrNotFound) shape they
// would see from a live 404.
func notFoundError(skillId uint32) error {
	return fmt.Errorf("skill %d not found: %w", skillId, requests.ErrNotFound)
}

// get returns a non-expired entry. Expired entries are treated as misses
// and overwritten in place by the subsequent refetch (lazy expiry — no
// sweeper; population is O(distinct templates)).
func (c *skillCache) get(t tenant.Model, skillId uint32) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tm, ok := c.perTenant[t.Id()]
	if !ok {
		return cacheEntry{}, false
	}
	e, ok := tm[skillId]
	if !ok || time.Now().After(e.expiresAt) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *skillCache) put(t tenant.Model, skillId uint32, e cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tm, ok := c.perTenant[t.Id()]
	if !ok {
		tm = map[uint32]cacheEntry{}
		c.perTenant[t.Id()] = tm
	}
	tm[skillId] = e
}

// SeedForTest installs a positive cache entry directly, bypassing the
// upstream fetch. Used by writer/handler tests to preload skill templates
// without a live REST call to atlas-data (mirrors the pre-task-122
// GetCache().Put seam). Only call from tests.
func SeedForTest(t tenant.Model, skillId uint32, m Model) {
	getSkillCache().put(t, skillId, cacheEntry{m: m, expiresAt: time.Now().Add(defaultTTL)})
}

// EvictTenant drops the tenant's cached skill templates. Invoked by
// listener.RegisterEvictor in main.go.
func EvictTenant(tid uuid.UUID) {
	c := getSkillCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.perTenant, tid)
}

// getByIdCached is the read-through used by Processor.GetById.
func getByIdCached(l logrus.FieldLogger, ctx context.Context, skillId uint32) (Model, error) {
	c := getSkillCache()
	if !c.cfg.enabled {
		return upstreamFn(l, ctx, skillId)
	}
	t := tenant.MustFromContext(ctx)
	if e, ok := c.get(t, skillId); ok {
		if e.negative {
			recordCache(t, outcomeCacheNegativeHit)
			return Model{}, notFoundError(skillId)
		}
		recordCache(t, outcomeCacheHit)
		return e.m, nil
	}
	recordCache(t, outcomeCacheMiss)
	m, err := upstreamFn(l, ctx, skillId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) && c.cfg.negativeTTL > 0 {
			c.put(t, skillId, cacheEntry{negative: true, expiresAt: time.Now().Add(c.cfg.negativeTTL)})
		}
		return Model{}, err
	}
	c.put(t, skillId, cacheEntry{m: m, expiresAt: time.Now().Add(c.cfg.ttl)})
	return m, nil
}
