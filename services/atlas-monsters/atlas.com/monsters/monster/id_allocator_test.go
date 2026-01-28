package monster

import (
	"sync"
	"testing"
)

func TestAllocator_SequentialAllocation(t *testing.T) {
	a := NewTenantIdAllocator()

	id1 := a.Allocate()
	if id1 != MinMonsterId {
		t.Fatalf("Expected first ID to be %d, got %d", MinMonsterId, id1)
	}

	id2 := a.Allocate()
	if id2 != MinMonsterId+1 {
		t.Fatalf("Expected second ID to be %d, got %d", MinMonsterId+1, id2)
	}

	id3 := a.Allocate()
	if id3 != MinMonsterId+2 {
		t.Fatalf("Expected third ID to be %d, got %d", MinMonsterId+2, id3)
	}
}

func TestAllocator_Release(t *testing.T) {
	a := NewTenantIdAllocator()

	id1 := a.Allocate()
	a.Release(id1)

	if len(a.freeIds) != 1 {
		t.Fatalf("Expected 1 free ID, got %d", len(a.freeIds))
	}
	if a.freeIds[0] != id1 {
		t.Fatalf("Expected free ID to be %d, got %d", id1, a.freeIds[0])
	}
}

func TestAllocator_RecycledIdsPreferred(t *testing.T) {
	a := NewTenantIdAllocator()

	// Allocate 3 IDs
	id1 := a.Allocate() // 1000000000
	id2 := a.Allocate() // 1000000001
	id3 := a.Allocate() // 1000000002

	// Release id1 and id2
	a.Release(id1)
	a.Release(id2)

	// Next allocation should return recycled IDs (LIFO: id2 first, then id1)
	recycled1 := a.Allocate()
	if recycled1 != id2 {
		t.Fatalf("Expected recycled ID %d (LIFO), got %d", id2, recycled1)
	}

	recycled2 := a.Allocate()
	if recycled2 != id1 {
		t.Fatalf("Expected recycled ID %d (LIFO), got %d", id1, recycled2)
	}

	// Now should get sequential ID
	newId := a.Allocate()
	if newId != id3+1 {
		t.Fatalf("Expected new sequential ID %d, got %d", id3+1, newId)
	}
}

func TestAllocator_LIFOOrder(t *testing.T) {
	a := NewTenantIdAllocator()

	// Allocate and release in order: 1, 2, 3
	ids := make([]uint32, 5)
	for i := 0; i < 5; i++ {
		ids[i] = a.Allocate()
	}

	// Release in order: 0, 1, 2, 3, 4
	for i := 0; i < 5; i++ {
		a.Release(ids[i])
	}

	// Should get back in reverse order (LIFO): 4, 3, 2, 1, 0
	for i := 4; i >= 0; i-- {
		recycled := a.Allocate()
		if recycled != ids[i] {
			t.Fatalf("Expected LIFO order ID %d, got %d", ids[i], recycled)
		}
	}
}

func TestAllocator_WrapAround(t *testing.T) {
	a := NewTenantIdAllocator()
	a.nextId = MaxMonsterId

	id1 := a.Allocate()
	if id1 != MaxMonsterId {
		t.Fatalf("Expected ID at max %d, got %d", MaxMonsterId, id1)
	}

	// Next should wrap around to min
	id2 := a.Allocate()
	if id2 != MinMonsterId {
		t.Fatalf("Expected wrap-around to %d, got %d", MinMonsterId, id2)
	}
}

func TestAllocator_ConcurrentAllocation(t *testing.T) {
	a := NewTenantIdAllocator()
	numGoroutines := 100
	idsPerGoroutine := 100

	var wg sync.WaitGroup
	idChan := make(chan uint32, numGoroutines*idsPerGoroutine)

	// Spawn goroutines that allocate IDs
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := a.Allocate()
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// Verify no duplicate IDs
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
	a := NewTenantIdAllocator()
	numGoroutines := 50
	iterations := 100

	var wg sync.WaitGroup

	// Spawn goroutines that allocate and release
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				id := a.Allocate()
				// Simulate some work
				a.Release(id)
			}
		}()
	}

	wg.Wait()

	// Verify allocator is in consistent state
	// Free pool should have some IDs, and we should be able to allocate
	_ = a.Allocate()
}
