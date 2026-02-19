package blocked

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

func TestRegistry_Block_And_IsBlocked(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)

	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
}

func TestRegistry_IsBlocked_NotBlocked(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
}

func TestRegistry_Unblock(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))

	GetRegistry().Unblock(ctx, 1000, 100000000, 1)
	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
}

func TestRegistry_Unblock_NonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().Unblock(ctx, 1000, 100000000, 1)
}

func TestRegistry_MultiplePortals(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	GetRegistry().Block(ctx, 1000, 100000000, 2)
	GetRegistry().Block(ctx, 1000, 200000000, 1)

	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 2))
	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 200000000, 1))
	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 200000000, 2))
}

func TestRegistry_ClearForCharacter(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	GetRegistry().Block(ctx, 1000, 100000000, 2)
	GetRegistry().Block(ctx, 1000, 200000000, 1)

	GetRegistry().ClearForCharacter(ctx, 1000)

	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 2))
	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 200000000, 1))
}

func TestRegistry_ClearForCharacter_NonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().ClearForCharacter(ctx, 9999)
}

func TestRegistry_ClearForCharacter_DoesNotAffectOthers(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	GetRegistry().Block(ctx, 2000, 100000000, 1)

	GetRegistry().ClearForCharacter(ctx, 1000)

	assert.False(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))
	assert.True(t, GetRegistry().IsBlocked(ctx, 2000, 100000000, 1))
}

func TestRegistry_GetForCharacter(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	GetRegistry().Block(ctx, 1000, 200000000, 2)

	result := GetRegistry().GetForCharacter(ctx, 1000)
	assert.Len(t, result, 2)

	portals := make(map[string]bool)
	for _, m := range result {
		portals[portalKey(m.MapId(), m.PortalId())] = true
	}
	assert.True(t, portals["100000000:1"])
	assert.True(t, portals["200000000:2"])
}

func TestRegistry_GetForCharacter_Empty(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	result := GetRegistry().GetForCharacter(ctx, 9999)
	assert.Empty(t, result)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	GetRegistry().Block(ctx1, 1000, 100000000, 1)

	assert.True(t, GetRegistry().IsBlocked(ctx1, 1000, 100000000, 1))
	assert.False(t, GetRegistry().IsBlocked(ctx2, 1000, 100000000, 1))
}

func TestRegistry_TenantIsolation_ClearDoesNotCrosstenants(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	GetRegistry().Block(ctx1, 1000, 100000000, 1)
	GetRegistry().Block(ctx2, 1000, 100000000, 1)

	GetRegistry().ClearForCharacter(ctx1, 1000)

	assert.False(t, GetRegistry().IsBlocked(ctx1, 1000, 100000000, 1))
	assert.True(t, GetRegistry().IsBlocked(ctx2, 1000, 100000000, 1))
}

func TestRegistry_BlockIdempotent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	GetRegistry().Block(ctx, 1000, 100000000, 1)
	GetRegistry().Block(ctx, 1000, 100000000, 1)

	assert.True(t, GetRegistry().IsBlocked(ctx, 1000, 100000000, 1))

	result := GetRegistry().GetForCharacter(ctx, 1000)
	assert.Len(t, result, 1)
}
