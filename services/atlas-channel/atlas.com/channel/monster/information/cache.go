package information

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// In-process, tenant-scoped TTL cache fronting GetById. Semantics ported
// from the task-060 Redis-backed cache in atlas-monsters
// (services/atlas-monsters/.../monster/information/cache.go) but
// memory-backed per the task-120 PRD user decision (no Redis hop on the
// movement hot path). Concurrent same-key misses may duplicate the upstream
// fetch (no singleflight) — bounded by template count, matches task-060.

const (
	envEnabled     = "MONSTER_INFO_CACHE_ENABLED"
	envTTL         = "MONSTER_INFO_CACHE_TTL"
	envNegativeTTL = "MONSTER_INFO_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute
)

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

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
	model     Model
	negative  bool
	expiresAt time.Time
}

type infoCache struct {
	cfg       cacheConfig
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]cacheEntry
}

var (
	infoCacheOnce sync.Once
	infoCachePtr  *infoCache
)

func getInfoCache() *infoCache {
	infoCacheOnce.Do(func() {
		infoCachePtr = &infoCache{
			cfg:       loadConfig(),
			perTenant: map[uuid.UUID]map[uint32]cacheEntry{},
		}
	})
	return infoCachePtr
}

// lookup returns a non-expired entry. Expired entries are treated as misses
// and overwritten in place by the subsequent refetch (lazy expiry — no
// sweeper; population is O(distinct templates)).
func (c *infoCache) lookup(tid uuid.UUID, monsterId uint32, now time.Time) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		return cacheEntry{}, false
	}
	e, ok := tenantMap[monsterId]
	if !ok || now.After(e.expiresAt) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *infoCache) put(tid uuid.UUID, monsterId uint32, e cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		tenantMap = map[uint32]cacheEntry{}
		c.perTenant[tid] = tenantMap
	}
	tenantMap[monsterId] = e
}

// EvictTenant drops every cached template entry for the tenant. Invoked by
// listener.RegisterEvictor in main.go.
func EvictTenant(tid uuid.UUID) {
	c := getInfoCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.perTenant, tid)
}

// upstreamFn is the test-overridable upstream fetch (task-060 precedent).
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(monsterId), Extract)()
}

// notFoundError synthesizes a not-found error for negative-cache hits so
// callers see the same errors.Is(err, requests.ErrNotFound) shape they
// would see from a live 404.
func notFoundError(monsterId uint32) error {
	return fmt.Errorf("monster %d not found: %w", monsterId, requests.ErrNotFound)
}
