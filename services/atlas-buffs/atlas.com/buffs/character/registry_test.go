package character

import (
	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	applied, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes, false, false, "")

	assert.NoError(t, err)
	assert.Len(t, applied, 1)
	b := applied[0]
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

	_, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes, false, false, "")
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

	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), duration, changes, false, false, "")

	cancelled, err := GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.Equal(t, sourceId, cancelled[0].SourceId())

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

	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001001), byte(5), int32(60), changes, false, false, "")
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001002), byte(5), int32(120), changes, false, false, "")
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2001003), byte(5), int32(180), changes, false, false, "")

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 3)

	GetRegistry().Cancel(ctx, characterId, int32(2001002))

	m, err = GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 2)

	buffs := m.Buffs()
	_, exists1 := buffs[srcKey(2001001)]
	_, exists2 := buffs[srcKey(2001002)]
	_, exists3 := buffs[srcKey(2001003)]
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

	_, _ = GetRegistry().Apply(ctx1, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(60), changes, false, false, "")

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

	_, _ = GetRegistry().Apply(ctx1, world.Id(0), channel.Id(0), 1000, int32(2001001), byte(5), int32(60), changes, false, false, "")
	_, _ = GetRegistry().Apply(ctx2, world.Id(0), channel.Id(0), 2000, int32(2001002), byte(5), int32(60), changes, false, false, "")

	tenants, err := GetRegistry().GetTenants(context.Background())
	assert.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestRegistry_GetCharacters(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, int32(2001001), byte(5), int32(60), changes, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 2000, int32(2001002), byte(5), int32(60), changes, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 3000, int32(2001003), byte(5), int32(60), changes, false, false, "")

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
			_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), changes, false, false, "")
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
			_, _ = GetRegistry().Apply(ctx1, world.Id(0), channel.Id(0), uint32(1000+idx), int32(2001000+idx), byte(5), int32(60), changes, false, false, "")
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = GetRegistry().Apply(ctx2, world.Id(0), channel.Id(0), uint32(1000+idx), int32(2001000+idx), byte(5), int32(60), changes, false, false, "")
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

	b1, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(60), changes, false, false, "")
	assert.NoError(t, err)
	assert.Len(t, b1, 1)
	assert.Equal(t, int32(60), b1[0].Duration())

	b2, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(5), int32(120), changes, false, false, "")
	assert.NoError(t, err)
	assert.Len(t, b2, 1)
	assert.Equal(t, int32(120), b2[0].Duration())

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	assert.Equal(t, int32(120), m.Buffs()[srcKey(sourceId)].Duration())
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
		_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), changes, false, false, "")
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
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), changes, false, false, "")

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
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), changes, false, false, "")

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
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2311003), byte(1), int32(60), holy, false, false, "")

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{"POISON": true})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.Equal(t, int32(124), cancelled[0].SourceId())

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsHoly := m.Buffs()[srcKey(2311003)]
	assert.True(t, holdsHoly)
}

func TestRegistry_CancelByStatTypes_MultiMatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	poison := []stat.Model{stat.NewStat("POISON", -10)}
	curse := []stat.Model{stat.NewStat("CURSE", -50)}
	weaken := []stat.Model{stat.NewStat("WEAKEN", -20)}
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(124), byte(1), int32(60), poison, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(125), byte(1), int32(60), curse, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(126), byte(1), int32(60), weaken, false, false, "")

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, uint32(1000), map[string]bool{
		"POISON": true,
		"CURSE":  true,
	})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 2)

	m, _ := GetRegistry().Get(ctx, uint32(1000))
	assert.Len(t, m.Buffs(), 1)
	_, holdsWeaken := m.Buffs()[srcKey(126)]
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

// --- accumulate mode (Beholder Hex) ---

// Different stats of the same source coexist (accumulate) and a source-wide
// Cancel removes them all.
func TestRegistry_Apply_Accumulate_DistinctStatsCoexist(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(1320009) // HEX_OF_THE_BEHOLDER

	a1, err := GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(25), int32(99000), []stat.Model{stat.NewStat("WEAPON_DEFENSE", 100)}, true, false, "")
	assert.NoError(t, err)
	assert.Len(t, a1, 1)
	_, err = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, sourceId, byte(25), int32(99000), []stat.Model{stat.NewStat("MAGIC_DEFENSE", 100)}, true, false, "")
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 2) // two independent per-stat entries under one source
	_, hasWdef := m.Buffs()[statKey(sourceId, "WEAPON_DEFENSE")]
	_, hasMdef := m.Buffs()[statKey(sourceId, "MAGIC_DEFENSE")]
	assert.True(t, hasWdef)
	assert.True(t, hasMdef)

	_, err = GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	m, _ = GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0)
}

// Re-rolling the same stat overwrites just that stat's entry (timer refresh),
// it does not create a second entry.
func TestRegistry_Apply_Accumulate_SameStatRefreshes(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	characterId := uint32(1000)
	sourceId := int32(1320009)

	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(25), int32(60000), []stat.Model{stat.NewStat("WEAPON_DEFENSE", 100)}, true, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(25), int32(99000), []stat.Model{stat.NewStat("WEAPON_DEFENSE", 100)}, true, false, "")

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	assert.Equal(t, int32(99000), m.Buffs()[statKey(sourceId, "WEAPON_DEFENSE")].Duration())
}

// Each accumulate stat expires on its own timer: a short-lived stat is reaped by
// GetExpired while a long-lived stat of the same source remains.
func TestRegistry_Apply_Accumulate_PerStatExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	characterId := uint32(1000)
	sourceId := int32(1320009)

	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(25), int32(1), []stat.Model{stat.NewStat("WEAPON_DEFENSE", 100)}, true, false, "") // 1ms
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(25), int32(99000), []stat.Model{stat.NewStat("MAGIC_DEFENSE", 100)}, true, false, "")
	time.Sleep(10 * time.Millisecond)

	expired := GetRegistry().GetExpired(ctx, characterId)
	assert.Len(t, expired, 1)
	assert.Equal(t, "WEAPON_DEFENSE", expired[0].Changes()[0].Type())

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1) // magic defense survives
	_, hasMdef := m.Buffs()[statKey(sourceId, "MAGIC_DEFENSE")]
	assert.True(t, hasMdef)
}

// Cancel by sourceId must return EVERY per-stat accumulate buff under that source
// (Beholder Hex) so the caller emits one EXPIRED each — otherwise the un-returned
// stats' icons stay stuck on the client (removed from storage, never cancelled).
func TestRegistry_Cancel_Accumulate_ReturnsAllStats(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	characterId := uint32(1000)
	sourceId := int32(1320009)
	for _, st := range []string{"WEAPON_DEFENSE", "MAGIC_DEFENSE", "WEAPON_ATTACK"} {
		_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(25), int32(99000), []stat.Model{stat.NewStat(st, 50)}, true, false, "")
	}

	cancelled, err := GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	assert.Len(t, cancelled, 3) // all three per-stat buffs returned for EXPIRED emission

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0) // and all removed from storage
}

// Regression: default (non-accumulate) Apply still replaces the whole source on
// recast — a multi-stat buff does not accumulate.
func TestRegistry_Apply_DefaultReplacesWholeSource(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	characterId := uint32(1000)
	sourceId := int32(2001001)

	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), []stat.Model{stat.NewStat("STR", 10), stat.NewStat("DEX", 5)}, false, false, "")
	_, _ = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(5), int32(60), []stat.Model{stat.NewStat("STR", 20), stat.NewStat("DEX", 9)}, false, false, "")

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1) // single whole-source entry, overwritten
	b := m.Buffs()[srcKey(sourceId)]
	assert.Len(t, b.Changes(), 2)
}

func TestRegistry_TenantSetIsPrefixed(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	changes := setupTestChanges()

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(2001001), byte(5), int32(60), changes, false, false, "")
	assert.NoError(t, err)

	tenants, err := GetRegistry().GetTenants(context.Background())
	assert.NoError(t, err)
	if len(tenants) != 1 {
		t.Fatalf("GetTenants() = %d want 1", len(tenants))
	}
}

func setupComboBuff(t *testing.T, ctx context.Context, characterId uint32, sourceId int32) {
	t.Helper()
	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, sourceId, byte(20), int32(150000), changes, false, false, "")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func comboAmount(t *testing.T, ctx context.Context, characterId uint32, sourceId int32) int32 {
	t.Helper()
	m, err := GetRegistry().Get(ctx, characterId)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	b, ok := m.Buffs()[srcKey(sourceId)]
	if !ok {
		t.Fatalf("no buff under srcKey(%d)", sourceId)
	}
	for _, c := range b.Changes() {
		if c.Type() == "COMBO" {
			return c.Amount()
		}
	}
	t.Fatal("no COMBO stat on buff")
	return 0
}

func TestRegistry_UpdateStatValue_Increment(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))

	var got int32
	for _, c := range updated.Changes() {
		if c.Type() == "COMBO" {
			got = c.Amount()
		}
	}
	assert.Equal(t, int32(2), got)
}

func TestRegistry_UpdateStatValue_IncrementClampsAtCap(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	// 1 -> +2 (double orb) with cap 2 must land exactly on the cap, not past it.
	_, changed, _, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 2, Cap: 2})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_NoChangeAtCap(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, changed, _, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)
	assert.True(t, changed) // 1 -> 2

	// drive to cap 2, then verify at-cap increment is a no-op
	_, changed, _, err = GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 5, Cap: 2})
	assert.NoError(t, err)
	assert.False(t, changed, "already at/above cap must be a no-op")
	assert.Equal(t, int32(2), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_SetResets(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, _, _, _ = GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 4, Cap: 6})
	assert.Equal(t, int32(5), comboAmount(t, ctx, 1000, 1111002))

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationSet, Amount: 1, Cap: 0})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
	assert.Equal(t, int32(1), comboAmount(t, ctx, 1000, 1111002))
}

func TestRegistry_UpdateStatValue_SetSameValueNoOp(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationSet, Amount: 1, Cap: 0})
	assert.NoError(t, err)
	assert.False(t, changed, "SET to the current value must be a no-op")
	assert.False(t, created)
}

func TestRegistry_UpdateStatValue_NoOps(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	cases := []struct {
		name        string
		characterId uint32
		sourceId    int32
		statType    string
		operation   string
		amount      int32
	}{
		{"unknown character", 9999, 1111002, "COMBO", character2.StatOperationIncrement, 1},
		{"wrong sourceId", 1000, 11111001, "COMBO", character2.StatOperationIncrement, 1},
		{"wrong stat type", 1000, 1111002, "MORPH", character2.StatOperationIncrement, 1},
		{"unknown operation", 1000, 1111002, "COMBO", "MULTIPLY", 1},
		{"non-positive increment", 1000, 1111002, "COMBO", character2.StatOperationIncrement, 0},
		{"set below 1", 1000, 1111002, "COMBO", character2.StatOperationSet, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), tc.characterId,
				StatValueUpdate{SourceId: tc.sourceId, StatType: tc.statType, Operation: tc.operation, Amount: tc.amount, Cap: 6})
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.False(t, created)
		})
	}
	assert.Equal(t, int32(1), comboAmount(t, ctx, 1000, 1111002), "no-op paths must not mutate the value")
}

func TestRegistry_UpdateStatValue_ExpiredBuffNoOp(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, 1111002, byte(20), int32(1), changes, false, false, "")
	assert.NoError(t, err)
	time.Sleep(5 * time.Millisecond) // duration is 1ms; let it lapse

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)
	assert.False(t, changed, "expired buff must be a no-op")
	assert.False(t, created)
}

func TestRegistry_UpdateStatValue_PreservesTimestamps(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	setupComboBuff(t, ctx, 1000, 1111002)

	before, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	orig := before.Buffs()[srcKey(1111002)]

	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
	assert.True(t, updated.CreatedAt().Equal(orig.CreatedAt()), "createdAt must be unchanged")
	assert.True(t, updated.ExpiresAt().Equal(orig.ExpiresAt()), "expiresAt must be unchanged (buff must not extend)")
}

// TestRegistry_UpdateStatValue_CreateIfMissingCreatesNoExpiryBuff is the
// Energy Charge accumulation entry point: the very first qualifying hit
// arrives with no buff at all, and the channel cannot check for one without
// a REST read on the attack path (task-216 design.md §4.2).
func TestRegistry_UpdateStatValue_CreateIfMissingCreatesNoExpiryBuff(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
		})

	assert.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, created)
	assert.True(t, updated.NoExpiry())
	assert.Equal(t, byte(20), updated.Level())
	assert.Len(t, updated.Changes(), 1)
	assert.Equal(t, "ENERGY_CHARGE", updated.Changes()[0].Type())
	assert.Equal(t, int32(102), updated.Changes()[0].Amount())
}

// A create whose Amount already exceeds Cap stores the clamped value, so the
// created buff can never start out above the accumulation ceiling.
func TestRegistry_UpdateStatValue_CreateIfMissingClampsToCap(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	updated, _, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    99999, Cap: 10000, CreateIfMissing: true, Level: 20,
		})

	assert.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, int32(10000), updated.Changes()[0].Amount())
}

// The second hit increments the buff the first hit created, and reports
// created=false so the processor emits STAT_UPDATED rather than APPLIED.
func TestRegistry_UpdateStatValue_CreateIfMissingThenIncrements(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	u := StatValueUpdate{
		SourceId: 5110001, StatType: "ENERGY_CHARGE",
		Operation: character2.StatOperationIncrement,
		Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
	}
	_, _, _, _ = GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000, u)
	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000, u)

	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
	assert.Equal(t, int32(204), updated.Changes()[0].Amount())
}

// CreateIfMissing is opt-in: without it a missing buff stays the no-op Combo
// depends on. This is the Combo regression guard.
func TestRegistry_UpdateStatValue_MissingBuffWithoutCreateIsNoOp(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}

// CreateIfMissing only makes sense for INCREMENT; SET must not conjure a buff.
func TestRegistry_UpdateStatValue_CreateIfMissingIgnoredForSet(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: character2.StatOperationSet, Amount: 15000, CreateIfMissing: true, Level: 20})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}

// At the 15000 charged sentinel every further gain is a no-op, because the
// value is already at/above the 10000 accumulation cap. FR-2.5 is structural,
// not a guard.
func TestRegistry_UpdateStatValue_ChargedSentinelBlocksGain(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, 5110001, 20, 31000,
		[]stat.Model{stat.NewStat("ENERGY_CHARGE", 15000)}, false, false, "")
	assert.NoError(t, err)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: character2.StatOperationIncrement, Amount: 102, Cap: 10000, CreateIfMissing: true, Level: 20})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}

func TestRegistry_ApplyNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	changes := []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}
	applied, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(5211006), byte(1), 0, changes, false, true, "")

	assert.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.True(t, applied[0].NoExpiry())
	assert.False(t, applied[0].Expired())
}

// A reap pass must remove an already-expired finite buff sitting next to a
// no-expiry buff, and must keep the no-expiry buff (FR-2.3 regression).
func TestRegistry_GetExpiredKeepsNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1001)

	// Finite 1ms buff → expired after the sleep below.
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(2001001), byte(5), 1, setupTestChanges(), false, false, "")
	assert.NoError(t, err)
	// No-expiry beacon.
	_, err = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true, "")
	assert.NoError(t, err)

	time.Sleep(5 * time.Millisecond)
	expired := GetRegistry().GetExpired(ctx, characterId)

	assert.Len(t, expired, 1)
	assert.Equal(t, int32(2001001), expired[0].SourceId())

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_CancelByStatTypesRemovesNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1002)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true, "")
	assert.NoError(t, err)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, characterId, map[string]bool{"HOMING_BEACON": true})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.True(t, cancelled[0].NoExpiry())
}

// FR-3.4 evidence: whole-character cancel removes a no-expiry buff too.
func TestRegistry_CancelAllRemovesNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1003)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5220011), byte(10), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true, "")
	assert.NoError(t, err)

	all := GetRegistry().CancelAll(ctx, characterId)
	assert.Len(t, all, 1)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 0)
}
