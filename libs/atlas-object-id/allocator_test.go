package objectid

import (
	"context"
	"strconv"
	"testing"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (Allocator, *goredis.Client, tenant.Model, tenant.Model, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	te1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	te2, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	a := NewRedisAllocator(client)
	return a, client, te1, te2, func() {
		_ = client.Close()
		mr.Close()
	}
}

// primeCounter writes the counter directly so tests can exercise behavior near
// RecycleThreshold without looping 2B allocations.
func primeCounter(t *testing.T, client *goredis.Client, te tenant.Model, value uint32) {
	t.Helper()
	require.NoError(t, client.Set(context.Background(), counterKey(te), strconv.FormatUint(uint64(value), 10), 0).Err())
}

func TestAllocate_sequentialFromMin(t *testing.T) {
	a, _, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	for i := uint32(0); i < 5; i++ {
		got, err := a.Allocate(ctx, te)
		require.NoError(t, err)
		require.Equal(t, MinId+i, got, "iteration %d", i)
	}
}

func TestAllocate_tenantsIsolated(t *testing.T) {
	a, _, te1, te2, cleanup := setup(t)
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

func TestRelease_belowThresholdIsNoop(t *testing.T) {
	// Guards the regression that caused the client crash: a just-destroyed
	// reactor's oid must not be handed to the next-allocated drop while the
	// client still has the old object on screen.
	a, client, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	var ids []uint32
	for i := 0; i < 3; i++ {
		got, err := a.Allocate(ctx, te)
		require.NoError(t, err)
		ids = append(ids, got)
	}

	require.NoError(t, a.Release(ctx, te, ids[0]))
	require.NoError(t, a.Release(ctx, te, ids[1]))
	require.NoError(t, a.Release(ctx, te, ids[2]))

	// Free list should be empty -- release is a no-op below threshold.
	n, err := client.LLen(ctx, freeKey(te)).Result()
	require.NoError(t, err)
	require.Zero(t, n, "free list must stay empty while counter is below RecycleThreshold")

	// Next allocation continues counting up, never reusing a released id.
	got, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, MinId+3, got)
}

func TestRelease_aboveThresholdRecyclesLIFO(t *testing.T) {
	a, client, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	// Jump the counter to just past the threshold so Release starts pushing.
	primeCounter(t, client, te, RecycleThreshold)

	ids := []uint32{MinId + 100, MinId + 200, MinId + 300}
	require.NoError(t, a.Release(ctx, te, ids[0]))
	require.NoError(t, a.Release(ctx, te, ids[1]))
	require.NoError(t, a.Release(ctx, te, ids[2]))

	// LIFO: most recent release first.
	got1, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, ids[2], got1)

	got2, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, ids[1], got2)

	got3, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, ids[0], got3)

	// Free list drained; next allocation falls through to INCR.
	got4, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, RecycleThreshold+1, got4)
}

func TestAllocator_keysRespectEnvPrefix(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	prefix := atlasredis.KeyPrefix()

	gotNext := counterKey(tm)
	if want := prefix + ":oid:" + id.String() + ":next"; gotNext != want {
		t.Fatalf("counterKey = %q, want %q", gotNext, want)
	}

	gotFree := freeKey(tm)
	if want := prefix + ":oid:" + id.String() + ":free"; gotFree != want {
		t.Fatalf("freeKey = %q, want %q", gotFree, want)
	}
}

func TestClear_resetsTenant(t *testing.T) {
	a, client, te, _, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	// Force a state where Release actually populates the free list.
	primeCounter(t, client, te, RecycleThreshold)
	require.NoError(t, a.Release(ctx, te, RecycleThreshold-42))
	require.NoError(t, a.Clear(ctx, te))

	got, err := a.Allocate(ctx, te)
	require.NoError(t, err)
	require.Equal(t, MinId, got, "post-Clear allocation should restart at MinId, not return the released id")
}
