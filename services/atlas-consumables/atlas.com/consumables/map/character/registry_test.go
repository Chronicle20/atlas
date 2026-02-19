package character

import (
	"context"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

func TestRegistry_AddCharacter_And_GetMap(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	characterId := uint32(12345)
	f := field.NewBuilder(1, 2, 100000000).Build()

	GetRegistry().AddCharacter(ctx, characterId, f)

	result, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)
	assert.Equal(t, f.WorldId(), result.WorldId())
	assert.Equal(t, f.ChannelId(), result.ChannelId())
	assert.Equal(t, f.MapId(), result.MapId())
}

func TestRegistry_RemoveCharacter(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	characterId := uint32(12346)
	f := field.NewBuilder(1, 2, 100000000).Build()

	GetRegistry().AddCharacter(ctx, characterId, f)

	_, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)

	GetRegistry().RemoveCharacter(ctx, characterId)

	_, ok = GetRegistry().GetMap(ctx, characterId)
	assert.False(t, ok)
}

func TestRegistry_GetMap_NotFound(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	_, ok := GetRegistry().GetMap(ctx, uint32(99999999))
	assert.False(t, ok)
}

func TestRegistry_AddCharacter_OverwritesExisting(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	characterId := uint32(12347)
	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(2, 3, 200000000).Build()

	GetRegistry().AddCharacter(ctx, characterId, f1)
	GetRegistry().AddCharacter(ctx, characterId, f2)

	result, ok := GetRegistry().GetMap(ctx, characterId)
	assert.True(t, ok)
	assert.Equal(t, f2.WorldId(), result.WorldId())
	assert.Equal(t, f2.ChannelId(), result.ChannelId())
	assert.Equal(t, f2.MapId(), result.MapId())
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	baseCharacterId := uint32(50000)
	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			characterId := baseCharacterId + uint32(idx)
			f := field.NewBuilder(world.Id(idx%256), channel.Id(idx%20), _map.Id(100000000+idx)).Build()
			GetRegistry().AddCharacter(ctx, characterId, f)
			_, _ = GetRegistry().GetMap(ctx, characterId)
			GetRegistry().RemoveCharacter(ctx, characterId)
		}(i)
	}

	wg.Wait()
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f := field.NewBuilder(1, 2, 100000000).Build()
	GetRegistry().AddCharacter(ctx1, 1000, f)

	_, found1 := GetRegistry().GetMap(ctx1, 1000)
	assert.True(t, found1)

	_, found2 := GetRegistry().GetMap(ctx2, 1000)
	assert.False(t, found2)
}

func TestRegistry_TenantIsolation_SameCharacterId(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(2, 2, 200000000).Build()

	GetRegistry().AddCharacter(ctx1, 1000, f1)
	GetRegistry().AddCharacter(ctx2, 1000, f2)

	result1, found1 := GetRegistry().GetMap(ctx1, 1000)
	assert.True(t, found1)
	assert.Equal(t, f1.MapId(), result1.MapId())

	result2, found2 := GetRegistry().GetMap(ctx2, 1000)
	assert.True(t, found2)
	assert.Equal(t, f2.MapId(), result2.MapId())
}
