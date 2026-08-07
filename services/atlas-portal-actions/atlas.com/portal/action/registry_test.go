package action

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

func setupRegistryTest(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
	return mr
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return ten
}

func testCtx(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func nullLogger(t *testing.T) (logrus.FieldLogger, *logtest.Hook) {
	t.Helper()
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	return l, hook
}

func TestRegistry_Add_And_Get(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	pa := PendingAction{
		CharacterId:    1000,
		WorldId:        1,
		ChannelId:      2,
		FailureMessage: "test failure",
	}

	GetRegistry().Add(l, ctx, sagaId, pa)

	result, found := GetRegistry().Get(l, ctx, sagaId)
	assert.True(t, found)
	assert.Equal(t, uint32(1000), result.CharacterId)
	assert.Equal(t, pa.WorldId, result.WorldId)
	assert.Equal(t, pa.ChannelId, result.ChannelId)
	assert.Equal(t, "test failure", result.FailureMessage)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, hook := nullLogger(t)

	_, found := GetRegistry().Get(l, ctx, uuid.New())
	assert.False(t, found)

	// A clean cache miss is not a Redis error and must not be logged as one.
	for _, e := range hook.AllEntries() {
		assert.NotEqual(t, logrus.WarnLevel, e.Level, "a clean not-found must not log a warning")
	}
}

func TestRegistry_Remove(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(l, ctx, sagaId, pa)

	_, found := GetRegistry().Get(l, ctx, sagaId)
	assert.True(t, found)

	GetRegistry().Remove(l, ctx, sagaId)

	_, found = GetRegistry().Get(l, ctx, sagaId)
	assert.False(t, found)
}

func TestRegistry_Remove_NonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	// Should not panic
	GetRegistry().Remove(l, ctx, uuid.New())
}

func TestRegistry_Add_OverwritesExisting(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	pa1 := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2, FailureMessage: "first"}
	pa2 := PendingAction{CharacterId: 2000, WorldId: 3, ChannelId: 4, FailureMessage: "second"}

	GetRegistry().Add(l, ctx, sagaId, pa1)
	GetRegistry().Add(l, ctx, sagaId, pa2)

	result, found := GetRegistry().Get(l, ctx, sagaId)
	assert.True(t, found)
	assert.Equal(t, uint32(2000), result.CharacterId)
	assert.Equal(t, "second", result.FailureMessage)
}

func TestRegistry_MultipleSagas(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId1 := uuid.New()
	sagaId2 := uuid.New()

	pa1 := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 1}
	pa2 := PendingAction{CharacterId: 2000, WorldId: 2, ChannelId: 2}

	GetRegistry().Add(l, ctx, sagaId1, pa1)
	GetRegistry().Add(l, ctx, sagaId2, pa2)

	r1, found := GetRegistry().Get(l, ctx, sagaId1)
	assert.True(t, found)
	assert.Equal(t, uint32(1000), r1.CharacterId)

	r2, found := GetRegistry().Get(l, ctx, sagaId2)
	assert.True(t, found)
	assert.Equal(t, uint32(2000), r2.CharacterId)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)
	l, _ := nullLogger(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(l, ctx1, sagaId, pa)

	_, found1 := GetRegistry().Get(l, ctx1, sagaId)
	assert.True(t, found1)

	_, found2 := GetRegistry().Get(l, ctx2, sagaId)
	assert.False(t, found2)
}

func TestRegistry_EmptyFailureMessage(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(l, ctx, sagaId, pa)

	result, found := GetRegistry().Get(l, ctx, sagaId)
	assert.True(t, found)
	assert.Equal(t, "", result.FailureMessage)
}

func TestRegistry_AddWithTTL_RoundTrip(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	pa := PendingAction{
		CharacterId: 1000,
		WorldId:     1,
		ChannelId:   2,
		Kind:        KindWarp,
	}
	GetRegistry().AddWithTTL(l, ctx, sagaId, pa, 60*time.Second)

	got, found := GetRegistry().Get(l, ctx, sagaId)
	require.True(t, found)
	assert.Equal(t, KindWarp, got.Kind)
	assert.Equal(t, uint32(1000), got.CharacterId)
}

// An entry written by a pre-deploy replica decodes with Kind == "".
func TestRegistry_LegacyEntryHasEmptyKind(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, _ := nullLogger(t)

	sagaId := uuid.New()
	GetRegistry().Add(l, ctx, sagaId, PendingAction{
		CharacterId:    1000,
		WorldId:        1,
		ChannelId:      2,
		FailureMessage: "legacy",
	})

	got, found := GetRegistry().Get(l, ctx, sagaId)
	require.True(t, found)
	assert.Equal(t, "", got.Kind, "a legacy entry must decode with an empty Kind")
}

// The reviewer's blocking finding: AddWithTTL is the sole recovery path for a
// warp whose saga never lands (task-184's ENTER handler stops emitting
// EnableActions once a warp is dispatched). A dropped Redis write here must
// not be silent — it must be logged so an operator has a trace instead of a
// permanently frozen player with no evidence.
func TestRegistry_AddWithTTL_LogsOnRedisError(t *testing.T) {
	mr := setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, hook := nullLogger(t)

	mr.Close() // every subsequent Redis command errors

	sagaId := uuid.New()
	GetRegistry().AddWithTTL(l, ctx, sagaId, PendingAction{
		CharacterId: 1000,
		WorldId:     1,
		ChannelId:   2,
		Kind:        KindWarp,
	}, 60*time.Second)

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["transaction_id"] == sagaId.String() {
			found = true
		}
	}
	assert.True(t, found, "a failed AddWithTTL write must log a Warn with the transaction id")
}

// Same failure mode as AddWithTTL, for the no-TTL path used by transports.
func TestRegistry_Add_LogsOnRedisError(t *testing.T) {
	mr := setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, hook := nullLogger(t)

	mr.Close()

	sagaId := uuid.New()
	GetRegistry().Add(l, ctx, sagaId, PendingAction{
		CharacterId: 1000,
		WorldId:     1,
		ChannelId:   2,
		Kind:        KindTransport,
	})

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["transaction_id"] == sagaId.String() {
			found = true
		}
	}
	assert.True(t, found, "a failed Add write must log a Warn with the transaction id")
}

// Get must distinguish a genuine Redis error from a clean not-found: both
// return found == false (preserving caller behaviour), but only the error
// case logs.
func TestRegistry_Get_LogsOnRedisError(t *testing.T) {
	mr := setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, hook := nullLogger(t)

	mr.Close()

	sagaId := uuid.New()
	_, found := GetRegistry().Get(l, ctx, sagaId)
	assert.False(t, found, "a Redis error on Get must still report not-found to preserve caller behaviour")

	var loggedErr bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["transaction_id"] == sagaId.String() {
			loggedErr = true
		}
	}
	assert.True(t, loggedErr, "a Redis error on Get must be logged, unlike a clean not-found")
}

func TestRegistry_Remove_LogsOnRedisError(t *testing.T) {
	mr := setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	l, hook := nullLogger(t)

	mr.Close()

	sagaId := uuid.New()
	GetRegistry().Remove(l, ctx, sagaId)

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["transaction_id"] == sagaId.String() {
			found = true
		}
	}
	assert.True(t, found, "a failed Remove must log a Warn with the transaction id")
}
