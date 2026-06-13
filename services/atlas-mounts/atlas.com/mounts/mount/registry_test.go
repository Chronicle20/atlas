package mount

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func findEntry(entries []ActiveEntry, characterId uint32) (ActiveEntry, bool) {
	for _, e := range entries {
		if e.CharacterId == characterId {
			return e, true
		}
	}
	return ActiveEntry{}, false
}

func TestRegistry_Add_And_GetActive(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	c := MountRideContext{WorldId: 1, SkillId: 80001000, VehicleId: 1902000}
	assert.NoError(t, GetRegistry().Add(ctx, 1000, c))

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	e, ok := findEntry(result, 1000)
	assert.True(t, ok)
	assert.Equal(t, c.WorldId, e.Ctx.WorldId)
	assert.Equal(t, c.SkillId, e.Ctx.SkillId)
	assert.Equal(t, c.VehicleId, e.Ctx.VehicleId)
	assert.Equal(t, ten.Id(), e.Tenant.Id())
}

func TestRegistry_Remove(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	c := MountRideContext{WorldId: 1, SkillId: 80001000, VehicleId: 1902000}
	assert.NoError(t, GetRegistry().Add(ctx, 1000, c))
	assert.NoError(t, GetRegistry().Remove(ctx, 1000))

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestRegistry_GetActive_Empty(t *testing.T) {
	setupRegistryTest(t)

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestRegistry_MultipleCharacters(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	c1 := MountRideContext{WorldId: 1, SkillId: 80001000, VehicleId: 1902000}
	c2 := MountRideContext{WorldId: 1, SkillId: 80001001, VehicleId: 1902001}

	assert.NoError(t, GetRegistry().Add(ctx, 1000, c1))
	assert.NoError(t, GetRegistry().Add(ctx, 2000, c2))

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	e1, ok := findEntry(result, 1000)
	assert.True(t, ok)
	assert.Equal(t, c1.VehicleId, e1.Ctx.VehicleId)

	e2, ok := findEntry(result, 2000)
	assert.True(t, ok)
	assert.Equal(t, c2.VehicleId, e2.Ctx.VehicleId)
}

func TestRegistry_Overwrite(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	c1 := MountRideContext{WorldId: 1, SkillId: 80001000, VehicleId: 1902000}
	c2 := MountRideContext{WorldId: 1, SkillId: 80001001, VehicleId: 1902001}

	assert.NoError(t, GetRegistry().Add(ctx, 1000, c1))
	assert.NoError(t, GetRegistry().Add(ctx, 1000, c2))

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	e, ok := findEntry(result, 1000)
	assert.True(t, ok)
	assert.Equal(t, c2.SkillId, e.Ctx.SkillId)
	assert.Equal(t, c2.VehicleId, e.Ctx.VehicleId)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	c1 := MountRideContext{WorldId: 1, SkillId: 80001000, VehicleId: 1902000}
	c2 := MountRideContext{WorldId: 2, SkillId: 80001001, VehicleId: 1902001}

	assert.NoError(t, GetRegistry().Add(ctx1, 1000, c1))
	assert.NoError(t, GetRegistry().Add(ctx2, 2000, c2))

	result, err := GetRegistry().GetActive(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	e1, ok := findEntry(result, 1000)
	assert.True(t, ok)
	assert.Equal(t, ten1.Id(), e1.Tenant.Id())
	assert.Equal(t, c1.WorldId, e1.Ctx.WorldId)

	e2, ok := findEntry(result, 2000)
	assert.True(t, ok)
	assert.Equal(t, ten2.Id(), e2.Tenant.Id())
	assert.Equal(t, c2.WorldId, e2.Ctx.WorldId)
}

func TestRegistry_RemoveNonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not error.
	assert.NoError(t, GetRegistry().Remove(ctx, 9999))
}
