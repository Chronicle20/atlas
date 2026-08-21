package character

import (
	"atlas-buffs/buff/stat"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestGetPeriodicEntriesEmptyRegistry(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	entries, err := GetRegistry().GetPeriodicEntries(ctx)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// TestGetPeriodicEntriesIgnoresNonPeriodicStats: a buff made entirely of flat
// combat modifiers yields nothing for the tick path.
func TestGetPeriodicEntriesIgnoresNonPeriodicStats(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 2001001, 1, 600000,
		[]stat.Model{stat.NewStat("WEAPON_ATTACK", 30)}, false, false, "")
	require.NoError(t, err)

	entries, err := GetRegistry().GetPeriodicEntries(ctx)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// TestGetPeriodicEntriesYieldsEveryPeriodicStatOnOneBuff: the pre-task-214 scan
// broke after the first match; two periodic stats on one buff must yield two
// entries.
func TestGetPeriodicEntriesYieldsEveryPeriodicStatOnOneBuff(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 1311008, 1, 600000,
		[]stat.Model{
			stat.NewStat("DRAGON_BLOOD", 48),
			stat.NewStat("POISON", 25),
			stat.NewStat("WEAPON_ATTACK", 30),
		}, false, false, "")
	require.NoError(t, err)

	entries, err := GetRegistry().GetPeriodicEntries(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	// Sorted by (characterId, statType).
	assert.Equal(t, "DRAGON_BLOOD", entries[0].StatType)
	assert.Equal(t, int32(48), entries[0].Amount)
	assert.Equal(t, "POISON", entries[1].StatType)
	assert.Equal(t, int32(25), entries[1].Amount)
	assert.Equal(t, world.Id(0), entries[0].WorldId)
	assert.Equal(t, channel.Id(1), entries[0].ChannelId)
	assert.Equal(t, uint32(100), entries[0].CharacterId)
}

// TestGetPeriodicEntriesDedupesByMaxAmount: two live buffs carrying the same
// periodic stat collapse to one entry, deterministically the larger amount.
func TestGetPeriodicEntriesDedupesByMaxAmount(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5001, 1, 600000,
		[]stat.Model{stat.NewStat("POISON", 10)}, false, false, "")
	require.NoError(t, err)
	_, err = GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5002, 1, 600000,
		[]stat.Model{stat.NewStat("POISON", 25)}, false, false, "")
	require.NoError(t, err)

	entries, err := GetRegistry().GetPeriodicEntries(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int32(25), entries[0].Amount)
}

func TestGetPeriodicEntriesSkipsExpiredBuffs(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	// Duration is MILLISECONDS and must be > 0 (buff/model.go:145). Expired()
	// reads the real wall clock, so a 1ms buff plus a short sleep is the only
	// way to produce a lapsed buff.
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), 100, 5001, 1, 1,
		[]stat.Model{stat.NewStat("POISON", 25)}, false, false, "")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	entries, err := GetRegistry().GetPeriodicEntries(ctx)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestPeriodicTickStoreRoundTrip(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	key := TickKey{CharacterId: 100, StatType: "POISON"}
	_, ok := GetRegistry().GetPeriodicTick(ctx, key)
	assert.False(t, ok, "no entry before the first tick")

	at := time.Now().Truncate(time.Second)
	GetRegistry().UpdatePeriodicTick(ctx, key, at)

	got, ok := GetRegistry().GetPeriodicTick(ctx, key)
	require.True(t, ok)
	assert.True(t, at.Equal(got), "expected %v, got %v", at, got)

	GetRegistry().ClearPeriodicTick(ctx, key)
	_, ok = GetRegistry().GetPeriodicTick(ctx, key)
	assert.False(t, ok, "cleared entry must not read back")
}

// TestPeriodicTickStoreKeysByStatType: two effects on one character throttle
// independently (FR-2.2).
func TestPeriodicTickStoreKeysByStatType(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	poison := TickKey{CharacterId: 100, StatType: "POISON"}
	blood := TickKey{CharacterId: 100, StatType: "DRAGON_BLOOD"}

	GetRegistry().UpdatePeriodicTick(ctx, poison, time.Now())
	_, ok := GetRegistry().GetPeriodicTick(ctx, blood)
	assert.False(t, ok, "DRAGON_BLOOD must not read POISON's throttle")
}

// TestPeriodicTickStoreIsTenantScoped: same character id, two tenants.
func TestPeriodicTickStoreIsTenantScoped(t *testing.T) {
	setupTestRegistry(t)
	ctxA := setupTestContext(t, setupTestTenant(t))
	ctxB := setupTestContext(t, setupTestTenant(t))

	key := TickKey{CharacterId: 100, StatType: "POISON"}
	GetRegistry().UpdatePeriodicTick(ctxA, key, time.Now())

	_, ok := GetRegistry().GetPeriodicTick(ctxB, key)
	assert.False(t, ok, "tenant B must not see tenant A's throttle")
}

// TestClearPeriodicTicksForRemovesOnlyPeriodicStats.
func TestClearPeriodicTicksForRemovesOnlyPeriodicStats(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	poison := TickKey{CharacterId: 100, StatType: "POISON"}
	blood := TickKey{CharacterId: 100, StatType: "DRAGON_BLOOD"}
	other := TickKey{CharacterId: 200, StatType: "POISON"}
	now := time.Now()
	GetRegistry().UpdatePeriodicTick(ctx, poison, now)
	GetRegistry().UpdatePeriodicTick(ctx, blood, now)
	GetRegistry().UpdatePeriodicTick(ctx, other, now)

	GetRegistry().ClearPeriodicTicksFor(ctx, 100,
		[]stat.Model{stat.NewStat("POISON", 25), stat.NewStat("WEAPON_ATTACK", 30)},
		[]stat.Model{stat.NewStat("DRAGON_BLOOD", 48)},
	)

	_, ok := GetRegistry().GetPeriodicTick(ctx, poison)
	assert.False(t, ok)
	_, ok = GetRegistry().GetPeriodicTick(ctx, blood)
	assert.False(t, ok)
	_, ok = GetRegistry().GetPeriodicTick(ctx, other)
	assert.True(t, ok, "another character's throttle is untouched")
}
