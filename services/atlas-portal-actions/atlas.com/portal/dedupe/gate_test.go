package dedupe

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupGate(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	InitGate(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	t.Cleanup(func() { gate = nil })
	return mr
}

func gateCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func testKey() Key {
	return Key{CharacterId: 100, MapId: 200090510, Instance: uuid.Nil, PortalId: 3}
}

func nullLogger(t *testing.T) (logrus.FieldLogger, *logtest.Hook) {
	t.Helper()
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	return l, hook
}

// FR-3.1/FR-3.4: the first ENTER passes, an identical second one inside the
// window does not.
func TestGate_DropsDuplicateInsideWindow(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	assert.True(t, GetGate().Allow(l, ctx, testKey()), "first ENTER passes")
	assert.False(t, GetGate().Allow(l, ctx, testKey()), "identical ENTER inside the TTL is dropped")
}

// FR-3.4: after the TTL elapses the same key passes again. The lock is never
// released explicitly — TTL expiry IS the release.
func TestGate_AllowsAfterTTL(t *testing.T) {
	mr := setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	require.True(t, GetGate().Allow(l, ctx, testKey()))
	mr.FastForward(enterGateTTL + time.Second)
	assert.True(t, GetGate().Allow(l, ctx, testKey()), "the gate reopens once the TTL expires")
}

// Different key components are different gates.
func TestGate_DistinctKeysDoNotCollide(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	base := testKey()
	require.True(t, GetGate().Allow(l, ctx, base))

	other := base
	other.PortalId = 4
	assert.True(t, GetGate().Allow(l, ctx, other), "a different portal is a different gate")

	other = base
	other.CharacterId = 101
	assert.True(t, GetGate().Allow(l, ctx, other), "a different character is a different gate")

	other = base
	other.MapId = 200090500
	assert.True(t, GetGate().Allow(l, ctx, other), "a different map is a different gate")

	other = base
	other.Instance = uuid.New()
	assert.True(t, GetGate().Allow(l, ctx, other), "a different instance is a different gate")
}

// NFR multi-tenancy: two tenants with identical character/map/portal must not
// share a gate.
func TestGate_TenantIsolation(t *testing.T) {
	setupGate(t)
	l, _ := nullLogger(t)

	ctxA := gateCtx(t)
	ctxB := gateCtx(t) // a different tenant uuid

	require.True(t, GetGate().Allow(l, ctxA, testKey()))
	assert.True(t, GetGate().Allow(l, ctxB, testKey()),
		"a second tenant's identical ENTER must not be gated by the first")
}

// FR-3.6: a Redis failure fails OPEN. Losing Redis must not make every portal
// in the game unusable.
func TestGate_FailsOpenOnRedisError(t *testing.T) {
	mr := setupGate(t)
	ctx := gateCtx(t)
	l, _ := nullLogger(t)

	mr.Close() // every subsequent command errors
	assert.True(t, GetGate().Allow(l, ctx, testKey()), "a Redis error must not block portal traversal")
}

// FR-3.6: an uninitialised gate (unit tests, misconfigured startup) allows.
func TestGate_NilGateAllows(t *testing.T) {
	gate = nil
	ctx := gateCtx(t)
	l, _ := nullLogger(t)
	assert.True(t, GetGate().Allow(l, ctx, testKey()))
}

// FR-3.5: a dropped duplicate is logged at Debug with the key components.
func TestGate_LogsDroppedDuplicateAtDebug(t *testing.T) {
	setupGate(t)
	ctx := gateCtx(t)
	l, hook := nullLogger(t)

	require.True(t, GetGate().Allow(l, ctx, testKey()))
	require.False(t, GetGate().Allow(l, ctx, testKey()))

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.DebugLevel && e.Data["portal_id"] == uint32(3) {
			found = true
			assert.Equal(t, uint32(100), e.Data["character_id"])
			assert.NotEmpty(t, e.Data["tenant_id"])
		}
	}
	assert.True(t, found, "the drop must be logged at Debug with the key components")
}
