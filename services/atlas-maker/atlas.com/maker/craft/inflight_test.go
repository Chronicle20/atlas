package craft

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSecondCraftWhileInFlightIsRejected(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()

	assert.True(t, g.TryAcquire(tenantId, 1001))
	assert.False(t, g.TryAcquire(tenantId, 1001), "a second acquire for the same key while the first is in flight must fail")
}

func TestGuardReleasesOnTerminalEvent(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()

	require := assert.New(t)
	require.True(g.TryAcquire(tenantId, 1001))
	require.False(g.TryAcquire(tenantId, 1001))

	g.Release(tenantId, 1001)

	require.True(g.TryAcquire(tenantId, 1001), "a craft should succeed again once the terminal event releases the guard")
}

func TestGuardIsPerCharacterAndPerTenant(t *testing.T) {
	g := newInflightGuard()
	tenantA := uuid.New()
	tenantB := uuid.New()

	assert.True(t, g.TryAcquire(tenantA, 1001), "character A's craft should acquire cleanly")
	assert.True(t, g.TryAcquire(tenantA, 1002), "a craft in flight for character A must not block character B in the same tenant")
	assert.True(t, g.TryAcquire(tenantB, 1001), "a craft in flight for character 1001 in tenant A must not block the same character id under tenant B")
}

func TestGuardIsConcurrencySafe(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()

	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	acquired := 0

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if g.TryAcquire(tenantId, 2001) {
				mu.Lock()
				acquired++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, acquired, "exactly one concurrent TryAcquire for the same key must succeed")
}

// TestGuardTrackAndReleaseConcurrencySafe exercises Track's and Release's own
// locking under -race: many goroutines race Track-ing a fresh transaction id
// onto a held entry concurrently with another goroutine releasing it, none of
// which should panic or corrupt inflightGuard's maps.
func TestGuardTrackAndReleaseConcurrencySafe(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()

	require := assert.New(t)
	require.True(g.TryAcquire(tenantId, 2001))

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			g.Track(tenantId, 2001, uuid.New())
		}()
		go func() {
			defer wg.Done()
			g.Release(tenantId, 2001)
		}()
	}
	wg.Wait()
}

func TestGuardReleasesByTransactionId(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()
	txId := uuid.New()

	require := assert.New(t)
	require.True(g.TryAcquire(tenantId, 1001))
	g.Track(tenantId, 1001, txId)

	g.ReleaseByTransactionId(tenantId, txId)

	require.True(g.TryAcquire(tenantId, 1001), "a craft should succeed again once the terminal event releases the guard by transaction id")
}

func TestGuardReleaseByUnknownTransactionIdIsNoOp(t *testing.T) {
	g := newInflightGuard()
	tenantId := uuid.New()
	txId := uuid.New()

	require := assert.New(t)
	require.True(g.TryAcquire(tenantId, 1001))
	g.Track(tenantId, 1001, txId)

	assert.NotPanics(t, func() {
		g.ReleaseByTransactionId(tenantId, uuid.New())
	})

	require.False(g.TryAcquire(tenantId, 1001), "an unknown transaction id must not release a held entry")
}
