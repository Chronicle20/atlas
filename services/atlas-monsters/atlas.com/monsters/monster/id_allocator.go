package monster

import (
	"sync"
)

const (
	// MinMonsterId is the minimum unique ID for monsters.
	MinMonsterId = uint32(1000000000)
	// MaxMonsterId is the maximum unique ID for monsters.
	MaxMonsterId = uint32(2000000000)
)

// TenantIdAllocator manages unique monster ID allocation for a single tenant.
// It provides O(1) allocation by maintaining a pool of recycled IDs and a sequential counter.
type TenantIdAllocator struct {
	nextId  uint32   // Next sequential ID to allocate
	freeIds []uint32 // Pool of recycled IDs (LIFO stack)
	mu      sync.Mutex
}

// NewTenantIdAllocator creates a new ID allocator starting at MinMonsterId.
func NewTenantIdAllocator() *TenantIdAllocator {
	return &TenantIdAllocator{
		nextId:  MinMonsterId,
		freeIds: make([]uint32, 0),
	}
}

// Allocate returns the next available monster ID.
// It prefers recycled IDs (LIFO order) over new sequential IDs.
func (a *TenantIdAllocator) Allocate() uint32 {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Prefer recycled IDs
	if len(a.freeIds) > 0 {
		id := a.freeIds[len(a.freeIds)-1]
		a.freeIds = a.freeIds[:len(a.freeIds)-1]
		return id
	}

	// Allocate new sequential ID
	id := a.nextId
	a.nextId++

	// Wrap around if exceeded max
	if a.nextId > MaxMonsterId {
		a.nextId = MinMonsterId
	}

	return id
}

// Release returns a monster ID to the free pool for reuse.
func (a *TenantIdAllocator) Release(id uint32) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.freeIds = append(a.freeIds, id)
}
