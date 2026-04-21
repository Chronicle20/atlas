package monster

import (
	"context"
	"sync"
	"testing"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
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

func TestAllocator_RecycledIdsPreferred(t *testing.T) {
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	id1 := a.Allocate(ctx, ten)
	id2 := a.Allocate(ctx, ten)
	_ = a.Allocate(ctx, ten) // id3

	a.Release(ctx, ten, id1)
	a.Release(ctx, ten, id2)

	// LIFO: id2 first, then id1.
	recycled1 := a.Allocate(ctx, ten)
	if recycled1 != id2 {
		t.Fatalf("Expected recycled ID %d (LIFO), got %d", id2, recycled1)
	}

	recycled2 := a.Allocate(ctx, ten)
	if recycled2 != id1 {
		t.Fatalf("Expected recycled ID %d (LIFO), got %d", id1, recycled2)
	}
}

func TestAllocator_LIFOOrder(t *testing.T) {
	testMiniRedis.FlushAll()
	ten := freshTenant(t)
	ctx := context.Background()
	a := GetIdAllocator()

	ids := make([]uint32, 5)
	for i := 0; i < 5; i++ {
		ids[i] = a.Allocate(ctx, ten)
	}

	for i := 0; i < 5; i++ {
		a.Release(ctx, ten, ids[i])
	}

	for i := 4; i >= 0; i-- {
		recycled := a.Allocate(ctx, ten)
		if recycled != ids[i] {
			t.Fatalf("Expected LIFO order ID %d, got %d", ids[i], recycled)
		}
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
