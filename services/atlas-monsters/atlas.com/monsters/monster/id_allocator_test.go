package monster

import (
	"context"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestAllocator_SequentialAllocation(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	a := GetIdAllocator()

	id1 := a.Allocate(ctx, ten)
	if id1 != MinMonsterId {
		t.Fatalf("Expected first ID to be %d, got %d", MinMonsterId, id1)
	}

	id2 := a.Allocate(ctx, ten)
	if id2 != MinMonsterId+1 {
		t.Fatalf("Expected second ID to be %d, got %d", MinMonsterId+1, id2)
	}

	id3 := a.Allocate(ctx, ten)
	if id3 != MinMonsterId+2 {
		t.Fatalf("Expected third ID to be %d, got %d", MinMonsterId+2, id3)
	}
}

func TestAllocator_RecycledIdsPreferred(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	a := GetIdAllocator()

	// Allocate 3 IDs
	id1 := a.Allocate(ctx, ten)
	id2 := a.Allocate(ctx, ten)
	_ = a.Allocate(ctx, ten) // id3

	// Release id1 and id2
	a.Release(ctx, ten, id1)
	a.Release(ctx, ten, id2)

	// Next allocation should return recycled IDs (LIFO: id2 first, then id1)
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
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	a := GetIdAllocator()

	ids := make([]uint32, 5)
	for i := 0; i < 5; i++ {
		ids[i] = a.Allocate(ctx, ten)
	}

	// Release in order: 0, 1, 2, 3, 4
	for i := 0; i < 5; i++ {
		a.Release(ctx, ten, ids[i])
	}

	// Should get back in reverse order (LIFO): 4, 3, 2, 1, 0
	for i := 4; i >= 0; i-- {
		recycled := a.Allocate(ctx, ten)
		if recycled != ids[i] {
			t.Fatalf("Expected LIFO order ID %d, got %d", ids[i], recycled)
		}
	}
}

func TestAllocator_ConcurrentAllocation(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
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
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
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

	// Verify allocator is in consistent state
	_ = a.Allocate(ctx, ten)
}
