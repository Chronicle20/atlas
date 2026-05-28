package character

import (
	"atlas-buffs/buff/stat"
	"context"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func setupTestChanges() []stat.Model {
	return []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
}

func TestRegistry_Apply(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes)

	assert.NoError(t, err)
	assert.Equal(t, sourceId, b.SourceId())
	assert.Equal(t, duration, b.Duration())
	assert.Len(t, b.Changes(), 2)
	assert.False(t, b.Expired())
}

func TestRegistry_Get(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	_, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Equal(t, characterId, m.Id())
	assert.Equal(t, worldId, m.WorldId())
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, err := GetRegistry().Get(ctx, 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_Cancel(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes)

	b, err := GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	assert.Equal(t, sourceId, b.SourceId())

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestRegistry_Cancel_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, err := GetRegistry().Cancel(ctx, 9999, 12345)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_MultipleBuffs(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	worldId := world.Id(0)
	characterId := uint32(1000)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001001), byte(5), int32(60), changes)
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001002), byte(5), int32(120), changes)
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001003), byte(5), int32(180), changes)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 3)

	GetRegistry().Cancel(ctx, characterId, int32(2001002))

	m, err = GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 2)

	buffs := m.Buffs()
	_, exists1 := buffs[int32(2001001)]
	_, exists2 := buffs[int32(2001002)]
	_, exists3 := buffs[int32(2001003)]
	assert.True(t, exists1)
	assert.False(t, exists2)
	assert.True(t, exists3)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := setupTestContext(t, ten1)
	ctx2 := setupTestContext(t, ten2)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx1, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(60), changes)

	m1, err := GetRegistry().Get(ctx1, characterId)
	assert.NoError(t, err)
	assert.Len(t, m1.Buffs(), 1)

	_, err = GetRegistry().Get(ctx2, characterId)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_GetTenants(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := setupTestContext(t, ten1)
	ctx2 := setupTestContext(t, ten2)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx1, world.Id(0), channel.Id(0), 1000, int32(2001001), byte(5), int32(60), changes)
	_, _ = GetRegistry().Apply(ctx2, world.Id(0), channel.Id(0), 2000, int32(2001002), byte(5), int32(60), changes)

	tenants, err := GetRegistry().GetTenants(context.Background())
	assert.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestRegistry_GetCharacters(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, int32(2001001), byte(5), int32(60), changes)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 2000, int32(2001002), byte(5), int32(60), changes)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 3000, int32(2001003), byte(5), int32(60), changes)

	chars := GetRegistry().GetCharacters(ctx)
	assert.Len(t, chars, 3)
}

func TestRegistry_ConcurrentApply(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			sourceId := int32(2001000 + idx)
			_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), changes)
		}(i)
	}

	wg.Wait()

	chars := GetRegistry().GetCharacters(ctx)
	assert.Len(t, chars, numGoroutines)
}

func TestRegistry_ConcurrentMultipleTenants(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := setupTestContext(t, ten1)
	ctx2 := setupTestContext(t, ten2)
	changes := setupTestChanges()

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = GetRegistry().Apply(ctx1, world.Id(0), channel.Id(0), uint32(1000+idx), int32(2001000+idx), byte(5), int32(60), changes)
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = GetRegistry().Apply(ctx2, world.Id(0), channel.Id(0), uint32(1000+idx), int32(2001000+idx), byte(5), int32(60), changes)
		}(i)
	}

	wg.Wait()

	chars1 := GetRegistry().GetCharacters(ctx1)
	chars2 := GetRegistry().GetCharacters(ctx2)

	assert.Len(t, chars1, 50)
	assert.Len(t, chars2, 50)
}

func TestRegistry_BuffReplacement(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)

	b1, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(60), changes)
	assert.NoError(t, err)
	assert.Equal(t, int32(60), b1.Duration())

	b2, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(120), changes)
	assert.NoError(t, err)
	assert.Equal(t, int32(120), b2.Duration())

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	assert.Equal(t, int32(120), m.Buffs()[sourceId].Duration())
}

func TestRegistry_ApplyAndCancel(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	characterId := uint32(1000)

	// Apply 50 buffs sequentially
	for i := 0; i < 50; i++ {
		sourceId := int32(2001000 + i)
		_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), changes)
	}

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 50)

	// Cancel first 25
	for i := 0; i < 25; i++ {
		sourceId := int32(2001000 + i)
		GetRegistry().Cancel(ctx, characterId, sourceId)
	}

	m, err = GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 25)
}

func TestRegistry_CancelByStatTypes_EmptyTypes(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	// Apply a POISON buff so we can prove an empty type set leaves it alone.
	changes := []stat.Model{stat.NewStat("POISON", -10)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), changes)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_CancelByStatTypes_NoMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	// Character has only HOLY_SYMBOL, ask to cancel POISON — should keep the buff.
	changes := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), changes)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_CancelByStatTypes_SingleMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	poison := []stat.Model{stat.NewStat("POISON", -10)}
	holy := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), holy)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.Equal(t, int32(124), cancelled[0].SourceId())

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsHoly := m.Buffs()[int32(2311003)]
	assert.True(t, holdsHoly)
}

func TestRegistry_CancelByStatTypes_MultiMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	poison := []stat.Model{stat.NewStat("POISON", -10)}
	curse := []stat.Model{stat.NewStat("CURSE", -50)}
	weaken := []stat.Model{stat.NewStat("WEAKEN", -20)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(125), byte(1), int32(60), curse)
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(126), byte(1), int32(60), weaken)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{
		"POISON": true,
		"CURSE":  true,
	})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 2)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsWeaken := m.Buffs()[int32(126)]
	assert.True(t, holdsWeaken)
}

func TestRegistry_CancelByStatTypes_UnknownCharacter(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(9999), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Nil(t, cancelled)
}

func TestRegistry_TenantSetIsPrefixed(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2001001), byte(5), int32(60), changes)
	assert.NoError(t, err)

	tenants, err := GetRegistry().GetTenants(context.Background())
	assert.NoError(t, err)
	if len(tenants) != 1 {
		t.Fatalf("GetTenants() = %d want 1", len(tenants))
	}
}
