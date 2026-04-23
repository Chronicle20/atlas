package objectid

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (Allocator, tenant.Model, tenant.Model, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	te1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	te2, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	a := NewRedisAllocator(client)
	return a, te1, te2, func() {
		_ = client.Close()
		mr.Close()
	}
}

func TestAllocate_sequentialFromMin(t *testing.T) {
	a, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	for i := uint32(0); i < 5; i++ {
		got, err := a.Allocate(ctx, te)
		require.NoError(t, err)
		require.Equal(t, MinId+i, got, "iteration %d", i)
	}
}

func TestAllocate_tenantsIsolated(t *testing.T) {
	a, te1, te2, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	// Allocate in te1 a few times.
	for i := 0; i < 3; i++ {
		_, err := a.Allocate(ctx, te1)
		require.NoError(t, err)
	}

	// te2's first allocation should still be MinId.
	got, err := a.Allocate(ctx, te2)
	require.NoError(t, err)
	require.Equal(t, MinId, got)
}

func TestRelease_recycledLIFO(t *testing.T) {
	a, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	// Allocate 3.
	var ids []uint32
	for i := 0; i < 3; i++ {
		got, err := a.Allocate(ctx, te)
		require.NoError(t, err)
		ids = append(ids, got)
	}

	// Release middle, then last. Free list head is last (LIFO).
	require.NoError(t, a.Release(ctx, te, ids[1]))
	require.NoError(t, a.Release(ctx, te, ids[2]))

	// Next two allocations should be [2], then [1].
	got1, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, ids[2], got1)

	got2, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, ids[1], got2)

	// Free list drained; next is counter+1.
	got3, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, MinId+3, got3)
}

func TestClear_resetsTenant(t *testing.T) {
	a, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := a.Allocate(ctx, te)
		require.NoError(t, err)
	}
	require.NoError(t, a.Release(ctx, te, 99))
	require.NoError(t, a.Clear(ctx, te))

	got, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, MinId, got, "post-Clear allocation should restart at MinId, not return the released 99")
}
