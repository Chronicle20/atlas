package character

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/field"
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

func TestRegistry_AddCharacter_And_GetLoggedIn(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(1, 1, 100000000).Build()
	GetRegistry().AddCharacter(ctx, 1000, f)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	mk, ok := result[1000]
	assert.True(t, ok)
	assert.Equal(t, f.MapId(), mk.Field.MapId())
}

func TestRegistry_RemoveCharacter(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(1, 1, 100000000).Build()
	GetRegistry().AddCharacter(ctx, 1000, f)
	GetRegistry().RemoveCharacter(ctx, 1000)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestRegistry_GetLoggedIn_Empty(t *testing.T) {
	setupRegistryTest(t)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestRegistry_MultipleCharacters(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 200000000).Build()

	GetRegistry().AddCharacter(ctx, 1000, f1)
	GetRegistry().AddCharacter(ctx, 2000, f2)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, f1.MapId(), result[1000].Field.MapId())
	assert.Equal(t, f2.MapId(), result[2000].Field.MapId())
}

func TestRegistry_OverwriteCharacter(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 200000000).Build()

	GetRegistry().AddCharacter(ctx, 1000, f1)
	GetRegistry().AddCharacter(ctx, 1000, f2)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, f2.MapId(), result[1000].Field.MapId())
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f := field.NewBuilder(1, 1, 100000000).Build()
	GetRegistry().AddCharacter(ctx1, 1000, f)
	GetRegistry().AddCharacter(ctx2, 2000, f)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	mk1, ok := result[1000]
	assert.True(t, ok)
	assert.Equal(t, ten1.Id(), mk1.Tenant.Id())

	mk2, ok := result[2000]
	assert.True(t, ok)
	assert.Equal(t, ten2.Id(), mk2.Tenant.Id())
}

func TestRegistry_CrossTenantGetLoggedIn(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(2, 1, 200000000).Build()

	GetRegistry().AddCharacter(ctx1, 1000, f1)
	GetRegistry().AddCharacter(ctx1, 1001, f1)
	GetRegistry().AddCharacter(ctx2, 2000, f2)

	result, err := GetRegistry().GetLoggedIn(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestRegistry_RemoveNonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().RemoveCharacter(ctx, 9999)
}
