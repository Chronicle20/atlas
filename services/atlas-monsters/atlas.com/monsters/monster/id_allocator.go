package monster

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// IdAllocator wraps the shared field-scoped object-id allocator so monsters,
// reactors, and drops share one oid namespace per field. The v83 client keys
// map objects by oid; colliding IDs across entity types crash the client.
type IdAllocator struct {
	inner objectid.Allocator
}

var (
	idAllocator     *IdAllocator
	idAllocatorOnce sync.Once
)

func InitIdAllocator(rc *goredis.Client) {
	idAllocatorOnce.Do(func() {
		idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}
	})
}

func GetIdAllocator() *IdAllocator {
	return idAllocator
}

// Allocate returns the next available oid for the given tenant. Prefers
// recycled IDs (LIFO) over new sequential values.
func (a *IdAllocator) Allocate(ctx context.Context, t tenant.Model) uint32 {
	id, err := a.inner.Allocate(ctx, t)
	if err != nil {
		// Preserve old fallback semantics: return MinId on failure rather than
		// propagating an error. Callers currently don't handle allocation failure.
		return objectid.MinId
	}
	return id
}

// Release returns an oid to the tenant's free pool for reuse.
func (a *IdAllocator) Release(ctx context.Context, t tenant.Model, id uint32) {
	_ = a.inner.Release(ctx, t, id)
}
