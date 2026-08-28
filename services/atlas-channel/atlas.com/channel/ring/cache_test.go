package ring

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

// resetRingCache resets the singleton for test isolation (pattern:
// resetInfoCache in monster/information/cache_test.go).
func resetRingCache() {
	ringCacheOnce = sync.Once{}
	ringCachePtr = nil
}

// TestRingCacheTenantIsolation proves the ring cache is scoped per tenant
// (PRD §8's multi-tenancy assertion), that a miss on an unknown character
// does not fall through to another character's entry, and that both
// EvictTenant and per-character invalidate drop only what they name.
func TestRingCacheTenantIsolation(t *testing.T) {
	t.Run("tenant isolation", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		c := getRingCache()

		tenantA, tenantB := uuid.New(), uuid.New()
		e1 := cacheEntry{halves: []Model{{cashId: 1}}}
		e2 := cacheEntry{halves: []Model{{cashId: 2}}}
		c.put(tenantA, 100, e1)
		c.put(tenantB, 100, e2)

		got, ok := c.lookup(tenantA, 100)
		if !ok || got.halves[0].cashId != 1 {
			t.Fatalf("lookup(tenantA, 100) = %+v, %v; want e1", got, ok)
		}
		got, ok = c.lookup(tenantB, 100)
		if !ok || got.halves[0].cashId != 2 {
			t.Fatalf("lookup(tenantB, 100) = %+v, %v; want e2", got, ok)
		}
	})

	t.Run("miss on unknown character", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		c := getRingCache()

		tenantA := uuid.New()
		c.put(tenantA, 100, cacheEntry{halves: []Model{{cashId: 1}}})

		if _, ok := c.lookup(tenantA, 101); ok {
			t.Fatal("lookup(tenantA, 101) = true, want false (unpopulated character)")
		}
	})

	t.Run("evict drops one tenant only", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		c := getRingCache()

		tenantA, tenantB := uuid.New(), uuid.New()
		c.put(tenantA, 100, cacheEntry{halves: []Model{{cashId: 1}}})
		c.put(tenantB, 100, cacheEntry{halves: []Model{{cashId: 2}}})

		EvictTenant(tenantA)

		if _, ok := c.lookup(tenantA, 100); ok {
			t.Fatal("lookup(tenantA, 100) = true after EvictTenant(tenantA), want false")
		}
		if _, ok := c.lookup(tenantB, 100); !ok {
			t.Fatal("lookup(tenantB, 100) = false after EvictTenant(tenantA), want true (other tenant untouched)")
		}
	})

	t.Run("invalidate drops one character only", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		c := getRingCache()

		tenantA := uuid.New()
		c.put(tenantA, 100, cacheEntry{halves: []Model{{cashId: 1}}})
		c.put(tenantA, 200, cacheEntry{halves: []Model{{cashId: 2}}})

		c.invalidate(tenantA, 100)

		if _, ok := c.lookup(tenantA, 100); ok {
			t.Fatal("lookup(tenantA, 100) = true after invalidate(tenantA, 100), want false")
		}
		if _, ok := c.lookup(tenantA, 200); !ok {
			t.Fatal("lookup(tenantA, 200) = false after invalidate(tenantA, 100), want true (other character untouched)")
		}
	})
}
