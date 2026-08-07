package door

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// IdAllocator wraps the shared field-scoped object-id allocator for doors.
// Unlike the monsters wrapper, Allocate surfaces errors so the door spawn can
// fail cleanly and release any already-allocated id. There is intentionally no
// MinId fallback here — a fallback would cause silent oid collisions (see
// TODO.md and the monster id_allocator known bug).
type IdAllocator struct{ inner objectid.Allocator }

var (
	idAllocator     *IdAllocator
	idAllocatorOnce sync.Once
)

func InitIdAllocator(rc *goredis.Client) {
	idAllocatorOnce.Do(func() { idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)} })
}

func GetIdAllocator() *IdAllocator { return idAllocator }

// Allocate returns (id, nil) or (0, err). Callers MUST fail the spawn on error
// and release any prior allocation — never substitute MinId (collision bug).
func (a *IdAllocator) Allocate(ctx context.Context, t tenant.Model) (uint32, error) {
	return a.inner.Allocate(ctx, t)
}

// Release returns an oid to the tenant's free pool. The underlying error is
// intentionally discarded — a failed release is non-fatal and the id will
// simply not be recycled.
func (a *IdAllocator) Release(ctx context.Context, t tenant.Model, id uint32) {
	_ = a.inner.Release(ctx, t, id)
}
