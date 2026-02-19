package action

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRegistryTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
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

func TestRegistry_Add_And_Get(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	pa := PendingAction{
		CharacterId:    1000,
		WorldId:        1,
		ChannelId:      2,
		FailureMessage: "test failure",
	}

	GetRegistry().Add(ctx, sagaId, pa)

	result, found := GetRegistry().Get(ctx, sagaId)
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

	_, found := GetRegistry().Get(ctx, uuid.New())
	assert.False(t, found)
}

func TestRegistry_Remove(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(ctx, sagaId, pa)

	_, found := GetRegistry().Get(ctx, sagaId)
	assert.True(t, found)

	GetRegistry().Remove(ctx, sagaId)

	_, found = GetRegistry().Get(ctx, sagaId)
	assert.False(t, found)
}

func TestRegistry_Remove_NonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().Remove(ctx, uuid.New())
}

func TestRegistry_Add_OverwritesExisting(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	pa1 := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2, FailureMessage: "first"}
	pa2 := PendingAction{CharacterId: 2000, WorldId: 3, ChannelId: 4, FailureMessage: "second"}

	GetRegistry().Add(ctx, sagaId, pa1)
	GetRegistry().Add(ctx, sagaId, pa2)

	result, found := GetRegistry().Get(ctx, sagaId)
	assert.True(t, found)
	assert.Equal(t, uint32(2000), result.CharacterId)
	assert.Equal(t, "second", result.FailureMessage)
}

func TestRegistry_MultipleSagas(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId1 := uuid.New()
	sagaId2 := uuid.New()

	pa1 := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 1}
	pa2 := PendingAction{CharacterId: 2000, WorldId: 2, ChannelId: 2}

	GetRegistry().Add(ctx, sagaId1, pa1)
	GetRegistry().Add(ctx, sagaId2, pa2)

	r1, found := GetRegistry().Get(ctx, sagaId1)
	assert.True(t, found)
	assert.Equal(t, uint32(1000), r1.CharacterId)

	r2, found := GetRegistry().Get(ctx, sagaId2)
	assert.True(t, found)
	assert.Equal(t, uint32(2000), r2.CharacterId)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(ctx1, sagaId, pa)

	_, found1 := GetRegistry().Get(ctx1, sagaId)
	assert.True(t, found1)

	_, found2 := GetRegistry().Get(ctx2, sagaId)
	assert.False(t, found2)
}

func TestRegistry_EmptyFailureMessage(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	sagaId := uuid.New()
	pa := PendingAction{CharacterId: 1000, WorldId: 1, ChannelId: 2}

	GetRegistry().Add(ctx, sagaId, pa)

	result, found := GetRegistry().Get(ctx, sagaId)
	assert.True(t, found)
	assert.Equal(t, "", result.FailureMessage)
}
