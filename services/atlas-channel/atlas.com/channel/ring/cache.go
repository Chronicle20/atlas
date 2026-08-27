package ring

import (
	"atlas-channel/listener"
	"sync"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// cacheEntry is the set of ring pair halves owned by one character,
// populated by Processor.Populate (task-269 task 12's entry point) and read
// by Processor.GetRingSet.
type cacheEntry struct {
	halves []Model
}

// ringCache is the per-tenant, per-character cache of ring pair halves,
// modeled directly on monster/information/cache.go:93-149's
// perTenant map[uuid.UUID]map[uint32]cacheEntry shape. Unlike that cache
// there is no TTL: population happens once at character load (PRD §8) and
// is invalidated explicitly (Processor.Invalidate) rather than expiring.
type ringCache struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]cacheEntry
}

var (
	ringCacheOnce sync.Once
	ringCachePtr  *ringCache
)

func getRingCache() *ringCache {
	ringCacheOnce.Do(func() {
		ringCachePtr = &ringCache{
			perTenant: map[uuid.UUID]map[uint32]cacheEntry{},
		}
	})
	return ringCachePtr
}

// lookup returns the cached halves for characterId, if any.
func (c *ringCache) lookup(tid uuid.UUID, characterId uint32) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		return cacheEntry{}, false
	}
	e, ok := tenantMap[characterId]
	return e, ok
}

func (c *ringCache) put(tid uuid.UUID, characterId uint32, e cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		tenantMap = map[uint32]cacheEntry{}
		c.perTenant[tid] = tenantMap
	}
	tenantMap[characterId] = e
}

// invalidate drops the cached halves for one character without touching any
// other character's entry in the same tenant.
func (c *ringCache) invalidate(tid uuid.UUID, characterId uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		return
	}
	delete(tenantMap, characterId)
}

// EvictTenant drops every cached ring entry for the tenant. Registered with
// listener.RegisterEvictor in this package's init() below.
func EvictTenant(tid uuid.UUID) {
	c := getRingCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.perTenant, tid)
}

// init registers EvictTenant so a tenant's ring cache is dropped once its
// last listener drains, per listener/evict.go:22-23 ("Safe to call from
// init() of any package that holds tenant-scoped state").
func init() {
	listener.RegisterEvictor(func(t tenant.Model) {
		EvictTenant(t.Id())
	})
}
