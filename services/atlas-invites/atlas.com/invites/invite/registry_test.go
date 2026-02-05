package invite

import (
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/stretchr/testify/assert"
)

func TestGetRegistry_Singleton(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()

	assert.Same(t, r1, r2, "GetRegistry should return the same instance")
}

func TestRegistry_Create(t *testing.T) {
	ten := setupTestTenant(t)

	m := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	assert.NotZero(t, m.Id())
	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(1001), m.OriginatorId())
	assert.Equal(t, world.Id(1), m.WorldId())
	assert.Equal(t, uint32(2001), m.TargetId())
	assert.Equal(t, "BUDDY", m.Type())
	assert.Equal(t, uint32(5001), m.ReferenceId())
	assert.WithinDuration(t, time.Now(), m.Age(), time.Second)
}

func TestRegistry_Create_IncrementingIds(t *testing.T) {
	ten := setupTestTenant(t)

	m1 := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	m2 := GetRegistry().Create(ten, 1002, 1, 2002, "PARTY", 5002)
	m3 := GetRegistry().Create(ten, 1003, 1, 2003, "GUILD", 5003)

	assert.Greater(t, m2.Id(), m1.Id())
	assert.Greater(t, m3.Id(), m2.Id())
}

func TestRegistry_Create_DuplicateReferenceId_ReturnsExisting(t *testing.T) {
	ten := setupTestTenant(t)

	m1 := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	m2 := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001) // Same referenceId

	assert.Equal(t, m1.Id(), m2.Id(), "Should return existing invite for duplicate referenceId")
}

func TestRegistry_Create_DifferentTypes_SameTarget(t *testing.T) {
	ten := setupTestTenant(t)

	m1 := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	m2 := GetRegistry().Create(ten, 1002, 1, 2001, "PARTY", 5002) // Same target, different type

	assert.NotEqual(t, m1.Id(), m2.Id(), "Should create separate invites for different types")
}

func TestRegistry_GetByOriginator(t *testing.T) {
	ten := setupTestTenant(t)
	created := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	found, err := GetRegistry().GetByOriginator(ten, 2001, "BUDDY", 1001)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), found.Id())
}

func TestRegistry_GetByOriginator_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	_, err := GetRegistry().GetByOriginator(ten, 2001, "BUDDY", 9999) // Wrong originator

	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())
}

func TestRegistry_GetByOriginator_WrongType(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	_, err := GetRegistry().GetByOriginator(ten, 2001, "PARTY", 1001) // Wrong type

	assert.Error(t, err)
}

func TestRegistry_GetByReference(t *testing.T) {
	ten := setupTestTenant(t)
	created := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	found, err := GetRegistry().GetByReference(ten, 2001, "BUDDY", 5001)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), found.Id())
}

func TestRegistry_GetByReference_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	_, err := GetRegistry().GetByReference(ten, 2001, "BUDDY", 9999) // Wrong reference

	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())
}

func TestRegistry_GetForCharacter(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	GetRegistry().Create(ten, 1002, 1, 2001, "PARTY", 5002)
	GetRegistry().Create(ten, 1003, 1, 2001, "GUILD", 5003)

	results, err := GetRegistry().GetForCharacter(ten, 2001)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestRegistry_GetForCharacter_Empty(t *testing.T) {
	ten := setupTestTenant(t)

	results, err := GetRegistry().GetForCharacter(ten, 9999)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestRegistry_GetForCharacter_OnlyReturnsTargetInvites(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	GetRegistry().Create(ten, 1002, 1, 3001, "BUDDY", 5002) // Different target

	results, err := GetRegistry().GetForCharacter(ten, 2001)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, uint32(2001), results[0].TargetId())
}

func TestRegistry_Delete(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	err := GetRegistry().Delete(ten, 2001, "BUDDY", 1001)

	assert.NoError(t, err)

	// Verify deleted
	_, err = GetRegistry().GetByOriginator(ten, 2001, "BUDDY", 1001)
	assert.Error(t, err)
}

func TestRegistry_Delete_NotFound(t *testing.T) {
	ten := setupTestTenant(t)

	err := GetRegistry().Delete(ten, 2001, "BUDDY", 9999)

	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())
}

func TestRegistry_Delete_OnlyDeletesMatching(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)
	GetRegistry().Create(ten, 1002, 1, 2001, "BUDDY", 5002)

	err := GetRegistry().Delete(ten, 2001, "BUDDY", 1001)

	assert.NoError(t, err)

	// Other invite should still exist
	results, _ := GetRegistry().GetForCharacter(ten, 2001)
	assert.Len(t, results, 1)
	assert.Equal(t, uint32(1002), results[0].OriginatorId())
}

func TestRegistry_GetExpired(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	// With a very long timeout, nothing should be expired
	results, err := GetRegistry().GetExpired(time.Hour)
	assert.NoError(t, err)

	// Filter results to our tenant
	var tenantResults []Model
	for _, r := range results {
		if r.Tenant() == ten {
			tenantResults = append(tenantResults, r)
		}
	}
	assert.Empty(t, tenantResults)
}

func TestRegistry_GetExpired_WithExpiredInvites(t *testing.T) {
	ten := setupTestTenant(t)
	GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	// With zero timeout, everything should be expired immediately
	time.Sleep(10 * time.Millisecond)
	results, err := GetRegistry().GetExpired(1 * time.Millisecond)
	assert.NoError(t, err)

	// Filter results to our tenant
	var tenantResults []Model
	for _, r := range results {
		if r.Tenant() == ten {
			tenantResults = append(tenantResults, r)
		}
	}
	assert.Len(t, tenantResults, 1)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)

	GetRegistry().Create(ten1, 1001, 1, 2001, "BUDDY", 5001)
	GetRegistry().Create(ten2, 1001, 1, 2001, "BUDDY", 5001)

	results1, _ := GetRegistry().GetForCharacter(ten1, 2001)
	results2, _ := GetRegistry().GetForCharacter(ten2, 2001)

	assert.Len(t, results1, 1)
	assert.Len(t, results2, 1)
	// Each tenant has its own ID counter, so IDs may be equal.
	// What matters is that tenants see only their own data.
	assert.Equal(t, ten1, results1[0].Tenant())
	assert.Equal(t, ten2, results2[0].Tenant())
}

func TestRegistry_TenantIsolation_GetByOriginator(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)

	GetRegistry().Create(ten1, 1001, 1, 2001, "BUDDY", 5001)

	// ten2 should not find ten1's invite
	_, err := GetRegistry().GetByOriginator(ten2, 2001, "BUDDY", 1001)
	assert.Error(t, err)
}

func TestRegistry_ConcurrentCreate(t *testing.T) {
	ten := setupTestTenant(t)

	var wg sync.WaitGroup
	numGoroutines := 10
	invitesPerGoroutine := 10

	results := make(chan Model, numGoroutines*invitesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineId int) {
			defer wg.Done()
			for j := 0; j < invitesPerGoroutine; j++ {
				originatorId := uint32(goroutineId*1000 + j + 1)   // +1 to avoid zero
				targetId := uint32(2000 + j + 1)                  // +1 to avoid zero
				referenceId := uint32(goroutineId*10000 + j + 1)  // +1 to avoid zero
				m := GetRegistry().Create(ten, originatorId, 1, targetId, "BUDDY", referenceId)
				results <- m
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect all results
	var allInvites []Model
	for m := range results {
		allInvites = append(allInvites, m)
	}

	assert.Len(t, allInvites, numGoroutines*invitesPerGoroutine)

	// Verify all IDs are unique
	idSet := make(map[uint32]bool)
	for _, m := range allInvites {
		assert.False(t, idSet[m.Id()], "duplicate ID found: %d", m.Id())
		idSet[m.Id()] = true
	}
}

func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	ten := setupTestTenant(t)

	// Pre-create some invites
	for i := 0; i < 5; i++ {
		GetRegistry().Create(ten, uint32(1000+i), 1, uint32(2000), "BUDDY", uint32(5000+i))
	}

	var wg sync.WaitGroup
	numReaders := 5
	numWriters := 3

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = GetRegistry().GetForCharacter(ten, 2000)
			}
		}()
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerId int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				GetRegistry().Create(ten, uint32(3000+writerId*100+j), 1, uint32(2000), "PARTY", uint32(6000+writerId*100+j))
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	results, err := GetRegistry().GetForCharacter(ten, 2000)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 5) // At least the pre-created ones
}

func TestRegistry_ConcurrentMultipleTenants(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)

	var wg sync.WaitGroup

	// Create invites for tenant 1
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			GetRegistry().Create(ten1, uint32(1000+i), 1, uint32(2000), "BUDDY", uint32(5000+i))
		}
	}()

	// Create invites for tenant 2
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			GetRegistry().Create(ten2, uint32(1000+i), 1, uint32(2000), "BUDDY", uint32(5000+i))
		}
	}()

	// Read from both tenants concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 30; i++ {
			_, _ = GetRegistry().GetForCharacter(ten1, 2000)
			_, _ = GetRegistry().GetForCharacter(ten2, 2000)
		}
	}()

	wg.Wait()

	// Verify tenant isolation
	results1, _ := GetRegistry().GetForCharacter(ten1, 2000)
	results2, _ := GetRegistry().GetForCharacter(ten2, 2000)

	for _, r := range results1 {
		assert.Equal(t, ten1, r.Tenant())
	}
	for _, r := range results2 {
		assert.Equal(t, ten2, r.Tenant())
	}
}
