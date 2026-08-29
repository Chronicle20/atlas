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
