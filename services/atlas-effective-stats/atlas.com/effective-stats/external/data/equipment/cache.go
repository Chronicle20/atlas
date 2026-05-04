package equipment

import (
	"context"
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EquipmentRequirements holds the six gating fields read from atlas-data.
type EquipmentRequirements struct {
	ReqLevel byte
	ReqJob   uint16 // v83 raw bitmask: 0=no restriction, 1=Warrior, 2=Mage, 4=Bowman, 8=Thief, 16=Pirate
	ReqStr   uint16
	ReqDex   uint16
	ReqInt   uint16
	ReqLuk   uint16
}

// Provider returns the requirements for a template id. The bool is false when
// the lookup failed and there is no cached value; callers MUST treat that as
// "this asset does not qualify for this evaluation".
type Provider func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool)

// fetcher is the indirection point for tests (Task 9 swaps it).
type fetcher func(ctx context.Context, l logrus.FieldLogger, templateId uint32) (EquipmentRequirements, error)

var defaultFetcher fetcher = func(ctx context.Context, l logrus.FieldLogger, templateId uint32) (EquipmentRequirements, error) {
	rm, err := RequestById(templateId)(l, ctx)
	if err != nil {
		return EquipmentRequirements{}, err
	}
	return EquipmentRequirements{
		ReqLevel: rm.ReqLevel,
		ReqJob:   rm.ReqJob,
		ReqStr:   rm.ReqStr,
		ReqDex:   rm.ReqDex,
		ReqInt:   rm.ReqInt,
		ReqLuk:   rm.ReqLuk,
	}, nil
}

type cache struct {
	mu    sync.RWMutex
	store map[uuid.UUID]map[uint32]EquipmentRequirements
}

var (
	once sync.Once
	inst *cache
)

func getCache() *cache {
	once.Do(func() {
		inst = &cache{store: make(map[uuid.UUID]map[uint32]EquipmentRequirements)}
	})
	return inst
}

func (c *cache) get(tID uuid.UUID, templateId uint32) (EquipmentRequirements, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.store[tID]
	if !ok {
		return EquipmentRequirements{}, false
	}
	r, ok := t[templateId]
	return r, ok
}

func (c *cache) put(tID uuid.UUID, templateId uint32, r EquipmentRequirements) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.store[tID]
	if !ok {
		t = make(map[uint32]EquipmentRequirements)
		c.store[tID] = t
	}
	t[templateId] = r
}

// reset is exposed package-internally for tests; production callers never
// invoke it.
func (c *cache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[uuid.UUID]map[uint32]EquipmentRequirements)
}

// SeedForTest pre-populates the cache for the current tenant with the given
// template requirements, bypassing the atlas-data fetch path. Intended for
// tests in other packages that exercise paths going through GetProvider but
// don't want to wire an HTTP fixture for atlas-data.
func SeedForTest(ctx context.Context, templateId uint32, r EquipmentRequirements) {
	t := tenant.MustFromContext(ctx)
	getCache().put(t.Id(), templateId, r)
}

// ResetCacheForTest clears the entire cache. Tests in other packages should
// call this in cleanup to avoid leaking state across runs.
func ResetCacheForTest() {
	getCache().reset()
}

// GetProvider returns a Provider closure bound to the given logger. The
// closure consults the per-tenant cache first; on cold-cache miss it fetches
// from atlas-data, caches success, and logs WARN on failure. Returning
// (_, false) means "treat this asset as unqualified for this evaluation".
func GetProvider(l logrus.FieldLogger) Provider {
	return func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool) {
		t := tenant.MustFromContext(ctx)
		if r, ok := getCache().get(t.Id(), templateId); ok {
			return r, true
		}
		r, err := defaultFetcher(ctx, l, templateId)
		if err != nil {
			l.WithError(err).Warnf("equipment template [%d] fetch failed; treating dependent assets as unqualified", templateId)
			return EquipmentRequirements{}, false
		}
		getCache().put(t.Id(), templateId, r)
		return r, true
	}
}
