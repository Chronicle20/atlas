package expression

import (
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_GetRegistry_Singleton(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()

	assert.Same(t, r1, r2, "GetRegistry should return the same instance")
}

func TestRegistry_Add(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	worldId := world.Id(0)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	characterId := uint32(1000)
	expr := uint32(5)

	m := r.add(ten, characterId, worldId, channelId, mapId, expr)

	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, characterId, m.CharacterId())
	assert.Equal(t, worldId, m.WorldId())
	assert.Equal(t, channelId, m.ChannelId())
	assert.Equal(t, mapId, m.MapId())
	assert.Equal(t, expr, m.Expression())
}

func TestRegistry_Add_SetsExpiration(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	before := time.Now()
	m := r.add(ten, 1000, world.Id(0), channel.Id(1), _map.Id(100000000), 5)
	after := time.Now()

	// Expiration should be approximately 5 seconds after creation
	expectedMin := before.Add(5 * time.Second)
	expectedMax := after.Add(5 * time.Second)

	assert.True(t, !m.Expiration().Before(expectedMin), "Expiration should be at least 5 seconds from before")
	assert.True(t, !m.Expiration().After(expectedMax), "Expiration should be at most 5 seconds from after")
}

func TestRegistry_Add_ReplacesExisting(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	characterId := uint32(1000)

	// Add first expression
	r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	// Add second expression for same character (should replace)
	m := r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 10)

	// Verify the new expression replaced the old one
	assert.Equal(t, uint32(10), m.Expression())

	// Verify only one expression exists
	retrieved, found := r.get(ten, characterId)
	assert.True(t, found)
	assert.Equal(t, uint32(10), retrieved.Expression())
}

func TestRegistry_Get(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	characterId := uint32(1000)
	r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	m, found := r.get(ten, characterId)

	assert.True(t, found)
	assert.Equal(t, characterId, m.CharacterId())
	assert.Equal(t, uint32(5), m.Expression())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	_, found := r.get(ten, 9999)

	assert.False(t, found)
}

func TestRegistry_Get_TenantNotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Don't add anything, just try to get
	_, found := r.get(ten, 1000)

	assert.False(t, found)
}

func TestRegistry_Clear(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	characterId := uint32(1000)
	r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	// Verify expression exists
	_, found := r.get(ten, characterId)
	assert.True(t, found)

	// Clear the expression
	r.clear(ten, characterId)

	// Verify expression is removed
	_, found = r.get(ten, characterId)
	assert.False(t, found)
}

func TestRegistry_Clear_NonExistent(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Clear should not panic for non-existent character
	r.clear(ten, 9999)
}

func TestRegistry_Clear_NonExistentTenant(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Clear should not panic for non-existent tenant
	r.clear(ten, 1000)
}

func TestRegistry_PopExpired(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Add an expression with immediate expiration (we'll manually set it)
	r.add(ten, 1000, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	// Manually set the expression to be expired by modifying the internal state
	r.lock.Lock()
	r.tenantLock[ten].Lock()
	if m, ok := r.expressionReg[ten][1000]; ok {
		// Create expired model
		expired := Model{
			tenant:      m.tenant,
			characterId: m.characterId,
			worldId:     m.worldId,
			channelId:   m.channelId,
			mapId:       m.mapId,
			expression:  m.expression,
			expiration:  time.Now().Add(-1 * time.Second), // Already expired
		}
		r.expressionReg[ten][1000] = expired
	}
	r.tenantLock[ten].Unlock()
	r.lock.Unlock()

	// Pop expired
	expired := r.popExpired()

	assert.Len(t, expired, 1)
	assert.Equal(t, uint32(1000), expired[0].CharacterId())

	// Verify it was removed from registry
	_, found := r.get(ten, 1000)
	assert.False(t, found)
}

func TestRegistry_PopExpired_LeavesNonExpired(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Add a non-expired expression (default 5 second expiration)
	r.add(ten, 1000, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	// Pop expired should return nothing
	expired := r.popExpired()

	assert.Len(t, expired, 0)

	// Verify expression still exists
	_, found := r.get(ten, 1000)
	assert.True(t, found)
}

func TestRegistry_PopExpired_MixedExpiration(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Add two expressions
	r.add(ten, 1000, world.Id(0), channel.Id(1), _map.Id(100000000), 5)
	r.add(ten, 2000, world.Id(0), channel.Id(1), _map.Id(100000000), 10)

	// Manually expire only one
	r.lock.Lock()
	r.tenantLock[ten].Lock()
	if m, ok := r.expressionReg[ten][1000]; ok {
		expired := Model{
			tenant:      m.tenant,
			characterId: m.characterId,
			worldId:     m.worldId,
			channelId:   m.channelId,
			mapId:       m.mapId,
			expression:  m.expression,
			expiration:  time.Now().Add(-1 * time.Second),
		}
		r.expressionReg[ten][1000] = expired
	}
	r.tenantLock[ten].Unlock()
	r.lock.Unlock()

	// Pop expired
	expiredList := r.popExpired()

	assert.Len(t, expiredList, 1)
	assert.Equal(t, uint32(1000), expiredList[0].CharacterId())

	// Verify non-expired still exists
	_, found := r.get(ten, 2000)
	assert.True(t, found)

	// Verify expired was removed
	_, found = r.get(ten, 1000)
	assert.False(t, found)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	characterId := uint32(1000)

	// Add expression in tenant1
	r.add(ten1, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 5)

	// Verify expression exists in tenant1
	m1, found1 := r.get(ten1, characterId)
	assert.True(t, found1)
	assert.Equal(t, uint32(5), m1.Expression())

	// Verify expression does not exist in tenant2
	_, found2 := r.get(ten2, characterId)
	assert.False(t, found2)
}

func TestRegistry_TenantIsolation_SameCharacterId(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	characterId := uint32(1000)

	// Add different expressions for same character ID in different tenants
	r.add(ten1, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 5)
	r.add(ten2, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), 10)

	// Verify tenant1 has expression 5
	m1, found1 := r.get(ten1, characterId)
	assert.True(t, found1)
	assert.Equal(t, uint32(5), m1.Expression())

	// Verify tenant2 has expression 10
	m2, found2 := r.get(ten2, characterId)
	assert.True(t, found2)
	assert.Equal(t, uint32(10), m2.Expression())
}

func TestRegistry_ConcurrentAdd(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), uint32(idx))
		}(i)
	}

	wg.Wait()

	// Verify all expressions were added
	for i := 0; i < numGoroutines; i++ {
		characterId := uint32(1000 + i)
		_, found := r.get(ten, characterId)
		assert.True(t, found, "Character %d should exist", characterId)
	}
}

func TestRegistry_ConcurrentAddAndClear(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	var wg sync.WaitGroup

	// Add expressions concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			r.add(ten, characterId, world.Id(0), channel.Id(1), _map.Id(100000000), uint32(idx))
		}(i)
	}

	wg.Wait()

	// Clear some expressions concurrently
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			r.clear(ten, characterId)
		}(i)
	}

	wg.Wait()

	// Verify first 25 are cleared
	for i := 0; i < 25; i++ {
		characterId := uint32(1000 + i)
		_, found := r.get(ten, characterId)
		assert.False(t, found, "Character %d should be cleared", characterId)
	}

	// Verify remaining 25 still exist
	for i := 25; i < 50; i++ {
		characterId := uint32(1000 + i)
		_, found := r.get(ten, characterId)
		assert.True(t, found, "Character %d should still exist", characterId)
	}
}

func TestRegistry_ConcurrentMultipleTenants(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	var wg sync.WaitGroup

	// Add expressions in tenant1 concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.add(ten1, uint32(1000+idx), world.Id(0), channel.Id(1), _map.Id(100000000), uint32(idx))
		}(i)
	}

	// Add expressions in tenant2 concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.add(ten2, uint32(1000+idx), world.Id(0), channel.Id(1), _map.Id(100000000), uint32(idx+100))
		}(i)
	}

	wg.Wait()

	// Verify tenant1 has correct expressions
	for i := 0; i < 50; i++ {
		m, found := r.get(ten1, uint32(1000+i))
		assert.True(t, found)
		assert.Equal(t, uint32(i), m.Expression())
	}

	// Verify tenant2 has correct expressions
	for i := 0; i < 50; i++ {
		m, found := r.get(ten2, uint32(1000+i))
		assert.True(t, found)
		assert.Equal(t, uint32(i+100), m.Expression())
	}
}

func TestRegistry_ResetForTesting(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := setupTestTenant(t)

	// Add some expressions
	r.add(ten, 1000, world.Id(0), channel.Id(1), _map.Id(100000000), 5)
	r.add(ten, 2000, world.Id(0), channel.Id(1), _map.Id(100000000), 10)

	// Verify they exist
	_, found1 := r.get(ten, 1000)
	_, found2 := r.get(ten, 2000)
	assert.True(t, found1)
	assert.True(t, found2)

	// Reset
	r.ResetForTesting()

	// Verify they're gone
	_, found1 = r.get(ten, 1000)
	_, found2 = r.get(ten, 2000)
	assert.False(t, found1)
	assert.False(t, found2)
}
