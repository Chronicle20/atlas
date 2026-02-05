package character

import (
	"atlas-buffs/buff/stat"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestChanges() []stat.Model {
	return []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
}

func TestRegistry_Apply(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b, err := r.Apply(ten, worldId, characterId, sourceId, duration, changes)

	assert.NoError(t, err)
	assert.Equal(t, sourceId, b.SourceId())
	assert.Equal(t, duration, b.Duration())
	assert.Len(t, b.Changes(), 2)
	assert.False(t, b.Expired())
}

func TestRegistry_Get(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	// Apply a buff first
	_, err := r.Apply(ten, worldId, characterId, sourceId, duration, changes)
	assert.NoError(t, err)

	// Get the character
	m, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, characterId, m.Id())
	assert.Equal(t, worldId, m.WorldId())
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	_, err := r.Get(ten, 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_Cancel(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	// Apply a buff first
	_, _ = r.Apply(ten, worldId, characterId, sourceId, duration, changes)

	// Cancel the buff
	b, err := r.Cancel(ten, characterId, sourceId)
	assert.NoError(t, err)
	assert.Equal(t, sourceId, b.SourceId())

	// Verify buff is removed
	m, _ := r.Get(ten, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestRegistry_Cancel_NotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Try to cancel non-existent buff
	_, err := r.Cancel(ten, 9999, 12345)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_MultipleBuffs(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	changes := setupTestChanges()

	// Apply multiple buffs
	_, _ = r.Apply(ten, worldId, characterId, int32(2001001), int32(60), changes)
	_, _ = r.Apply(ten, worldId, characterId, int32(2001002), int32(120), changes)
	_, _ = r.Apply(ten, worldId, characterId, int32(2001003), int32(180), changes)

	m, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 3)

	// Cancel one buff
	r.Cancel(ten, characterId, int32(2001002))

	m, err = r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 2)

	// Verify correct buffs remain
	buffs := m.Buffs()
	_, exists1 := buffs[int32(2001001)]
	_, exists2 := buffs[int32(2001002)]
	_, exists3 := buffs[int32(2001003)]
	assert.True(t, exists1)
	assert.False(t, exists2)
	assert.True(t, exists3)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)
	changes := setupTestChanges()

	// Apply buff in tenant1
	_, _ = r.Apply(ten1, worldId, characterId, sourceId, int32(60), changes)

	// Verify buff exists in tenant1
	m1, err := r.Get(ten1, characterId)
	assert.NoError(t, err)
	assert.Len(t, m1.Buffs(), 1)

	// Verify buff does not exist in tenant2
	_, err = r.Get(ten2, characterId)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_GetTenants(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	changes := setupTestChanges()

	// Apply buffs in both tenants
	_, _ = r.Apply(ten1, world.Id(0), 1000, int32(2001001), int32(60), changes)
	_, _ = r.Apply(ten2, world.Id(0), 2000, int32(2001002), int32(60), changes)

	tenants, err := r.GetTenants()
	assert.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestRegistry_GetCharacters(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)
	changes := setupTestChanges()

	// Apply buffs to multiple characters
	_, _ = r.Apply(ten, world.Id(0), 1000, int32(2001001), int32(60), changes)
	_, _ = r.Apply(ten, world.Id(0), 2000, int32(2001002), int32(60), changes)
	_, _ = r.Apply(ten, world.Id(0), 3000, int32(2001003), int32(60), changes)

	chars := r.GetCharacters(ten)
	assert.Len(t, chars, 3)
}

func TestRegistry_ConcurrentApply(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)
	changes := setupTestChanges()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			sourceId := int32(2001000 + idx)
			_, _ = r.Apply(ten, world.Id(0), characterId, sourceId, int32(60), changes)
		}(i)
	}

	wg.Wait()

	chars := r.GetCharacters(ten)
	assert.Len(t, chars, numGoroutines)
}

func TestRegistry_ConcurrentApplyAndCancel(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)
	changes := setupTestChanges()

	characterId := uint32(1000)

	var wg sync.WaitGroup

	// Apply buffs concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sourceId := int32(2001000 + idx)
			_, _ = r.Apply(ten, world.Id(0), characterId, sourceId, int32(60), changes)
		}(i)
	}

	wg.Wait()

	// Cancel some buffs concurrently
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sourceId := int32(2001000 + idx)
			r.Cancel(ten, characterId, sourceId)
		}(i)
	}

	wg.Wait()

	m, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 25)
}

func TestRegistry_ConcurrentMultipleTenants(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	changes := setupTestChanges()

	var wg sync.WaitGroup

	// Apply buffs in tenant1 concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = r.Apply(ten1, world.Id(0), uint32(1000+idx), int32(2001000+idx), int32(60), changes)
		}(i)
	}

	// Apply buffs in tenant2 concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = r.Apply(ten2, world.Id(0), uint32(1000+idx), int32(2001000+idx), int32(60), changes)
		}(i)
	}

	wg.Wait()

	chars1 := r.GetCharacters(ten1)
	chars2 := r.GetCharacters(ten2)

	assert.Len(t, chars1, 50)
	assert.Len(t, chars2, 50)
}

func TestRegistry_BuffReplacement(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)
	changes := setupTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	sourceId := int32(2001001)

	// Apply buff with 60 second duration
	b1, err := r.Apply(ten, worldId, characterId, sourceId, int32(60), changes)
	assert.NoError(t, err)
	assert.Equal(t, int32(60), b1.Duration())

	// Apply same source with different duration (should replace)
	b2, err := r.Apply(ten, worldId, characterId, sourceId, int32(120), changes)
	assert.NoError(t, err)
	assert.Equal(t, int32(120), b2.Duration())

	// Verify only one buff exists
	m, _ := r.Get(ten, characterId)
	assert.Len(t, m.Buffs(), 1)
	assert.Equal(t, int32(120), m.Buffs()[sourceId].Duration())
}
