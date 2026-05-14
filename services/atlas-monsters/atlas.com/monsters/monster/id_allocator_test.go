package monster

import (
	"context"
	"strconv"
	"sync"
	"testing"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func freshTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return te
}

func TestAllocator_SequentialAllocation(t *testing.T) {
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	id1 := a.Allocate(ctx, ten)
	if id1 != objectid.MinId {
		t.Fatalf("Expected first ID to be %d, got %d", objectid.MinId, id1)
	}

	id2 := a.Allocate(ctx, ten)
	if id2 != objectid.MinId+1 {
		t.Fatalf("Expected second ID to be %d, got %d", objectid.MinId+1, id2)
	}

	id3 := a.Allocate(ctx, ten)
	if id3 != objectid.MinId+2 {
		t.Fatalf("Expected third ID to be %d, got %d", objectid.MinId+2, id3)
	}
}

func TestAllocator_ReleaseIsNoopBelowThreshold(t *testing.T) {
	// Guards the client-crash regression: a recently released oid must not be
	// handed back out while the counter is still in the "fresh" range, because
	// the client may not yet have processed the Destroy packet for the old
	// object when the new one spawns with the same oid.
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	id1 := a.Allocate(ctx, ten)
	id2 := a.Allocate(ctx, ten)
	id3 := a.Allocate(ctx, ten)

	a.Release(ctx, ten, id1)
	a.Release(ctx, ten, id2)
	a.Release(ctx, ten, id3)

	// Next allocation should continue up, not revisit id1/id2/id3.
	next := a.Allocate(ctx, ten)
	if next != objectid.MinId+3 {
		t.Fatalf("Expected counter to advance to %d, got %d (possibly recycled a released id)", objectid.MinId+3, next)
	}
}

func TestAllocator_RecyclesLIFONearExhaustion(t *testing.T) {
	// Safety valve: once the counter has climbed past RecycleThreshold, released
	// oids should come back LIFO so the range doesn't get exhausted.
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	// Jump the counter to the threshold by writing directly to miniredis --
	// looping 2B allocations would be absurd.
	counterKeyName := atlasredis.KeyPrefix() + ":oid:" + ten.Id().String() + ":next"
	if err := testMiniRedis.Set(counterKeyName, strconv.FormatUint(uint64(objectid.RecycleThreshold), 10)); err != nil {
		t.Fatalf("prime counter: %v", err)
	}

	ids := []uint32{objectid.MinId + 10, objectid.MinId + 20, objectid.MinId + 30}
	for _, id := range ids {
		a.Release(ctx, ten, id)
	}

	// LIFO: last-released comes back first.
	for i := len(ids) - 1; i >= 0; i-- {
		got := a.Allocate(ctx, ten)
		if got != ids[i] {
			t.Fatalf("Expected LIFO recycled %d, got %d", ids[i], got)
		}
	}

	// Free list drained; fall through to INCR past the threshold.
	got := a.Allocate(ctx, ten)
	if got != objectid.RecycleThreshold+1 {
		t.Fatalf("Expected post-recycle allocation %d, got %d", objectid.RecycleThreshold+1, got)
	}
}

func TestAllocator_TenantsAreIsolated(t *testing.T) {
	testMiniRedis.FlushAll()
	ten1 := freshTenant(t)
	ten2 := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	_ = a.Allocate(ctx, ten1)
	_ = a.Allocate(ctx, ten1)

	first := a.Allocate(ctx, ten2)
	if first != objectid.MinId {
		t.Fatalf("Expected fresh tenant to start at %d, got %d", objectid.MinId, first)
	}
}

func TestAllocator_ConcurrentAllocation(t *testing.T) {
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	numGoroutines := 100
	idsPerGoroutine := 100

	var wg sync.WaitGroup
	idChan := make(chan uint32, numGoroutines*idsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := a.Allocate(ctx, ten)
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)

	seen := make(map[uint32]bool)
	for id := range idChan {
		if seen[id] {
			t.Fatalf("Duplicate ID allocated: %d", id)
		}
		seen[id] = true
	}

	expectedCount := numGoroutines * idsPerGoroutine
	if len(seen) != expectedCount {
		t.Fatalf("Expected %d unique IDs, got %d", expectedCount, len(seen))
	}
}

func TestAllocator_ConcurrentAllocateAndRelease(t *testing.T) {
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	numGoroutines := 50
	iterations := 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				id := a.Allocate(ctx, ten)
				a.Release(ctx, ten, id)
			}
		}()
	}

	wg.Wait()

	// Verify allocator is in a consistent state.
	_ = a.Allocate(ctx, ten)
}
