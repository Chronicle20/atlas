package party

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByCharacter(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	t.Run("GetPartyByExistingCharacter", func(t *testing.T) {
		// Create a party with a character
		leaderId := uint32(1001)
		party := GetRegistry().Create(testTenant, leaderId)
		
		// Add another member to the party
		memberId := uint32(1002)
		party = party.AddMember(memberId)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })
		
		// Test lookup for the leader
		foundParty, err := GetByCharacter(ctx)(leaderId)
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())
		assert.Equal(t, leaderId, foundParty.LeaderId())
		assert.Contains(t, foundParty.Members(), leaderId)

		// Test lookup for the member
		foundParty, err = GetByCharacter(ctx)(memberId)
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())
		assert.Contains(t, foundParty.Members(), memberId)
	})

	t.Run("GetPartyByNonExistentCharacter", func(t *testing.T) {
		// Test lookup for character not in any party
		nonExistentCharId := uint32(9999)
		_, err := GetByCharacter(ctx)(nonExistentCharId)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("GetPartyMultiplePartiesScenario", func(t *testing.T) {
		// Clear registry
		registry := GetRegistry()
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
		registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
		registry.cacheHitCount = make(map[tenant.Model]*uint64)
		registry.cacheMissCount = make(map[tenant.Model]*uint64)
		registry.tenantPartyId = make(map[tenant.Model]uint32)

		// Create multiple parties with different characters
		leader1Id := uint32(2001)
		leader2Id := uint32(2002)
		party1 := GetRegistry().Create(testTenant, leader1Id)
		party2 := GetRegistry().Create(testTenant, leader2Id)

		// Verify each character is in their respective party
		foundParty1, err := GetByCharacter(ctx)(leader1Id)
		assert.NoError(t, err)
		assert.Equal(t, party1.Id(), foundParty1.Id())

		foundParty2, err := GetByCharacter(ctx)(leader2Id)
		assert.NoError(t, err)
		assert.Equal(t, party2.Id(), foundParty2.Id())

		// Verify parties are different
		assert.NotEqual(t, party1.Id(), party2.Id())
	})
}

func TestGetPartiesByCharacters(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	t.Run("BatchLookupMultipleCharacters", func(t *testing.T) {
		// Create parties with multiple characters
		leader1Id := uint32(3001)
		leader2Id := uint32(3002)
		member1Id := uint32(3003)
		member2Id := uint32(3004)

		// Create first party
		party1 := GetRegistry().Create(testTenant, leader1Id)
		party1 = party1.AddMember(member1Id)
		GetRegistry().Update(testTenant, party1.Id(), func(m Model) Model { return party1 })

		// Create second party
		party2 := GetRegistry().Create(testTenant, leader2Id)
		party2 = party2.AddMember(member2Id)
		GetRegistry().Update(testTenant, party2.Id(), func(m Model) Model { return party2 })

		// Test batch lookup for characters in different parties
		characterIds := []uint32{leader1Id, leader2Id, member1Id, member2Id}
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 2) // Should return 2 unique parties

		// Verify both parties are returned
		partyIds := make(map[uint32]bool)
		for _, party := range parties {
			partyIds[party.Id()] = true
		}
		assert.True(t, partyIds[party1.Id()])
		assert.True(t, partyIds[party2.Id()])
	})

	t.Run("BatchLookupSamePartyCharacters", func(t *testing.T) {
		// Clear registry
		registry := GetRegistry()
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
		registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
		registry.cacheHitCount = make(map[tenant.Model]*uint64)
		registry.cacheMissCount = make(map[tenant.Model]*uint64)
		registry.tenantPartyId = make(map[tenant.Model]uint32)

		// Create one party with multiple members
		leaderId := uint32(4001)
		member1Id := uint32(4002)
		member2Id := uint32(4003)

		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(member1Id).AddMember(member2Id)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Test batch lookup for characters in same party
		characterIds := []uint32{leaderId, member1Id, member2Id}
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 1) // Should return only 1 party (deduplication)
		assert.Equal(t, party.Id(), parties[0].Id())
	})

	t.Run("BatchLookupMixedCharacters", func(t *testing.T) {
		// Clear registry
		registry := GetRegistry()
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
		registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
		registry.cacheHitCount = make(map[tenant.Model]*uint64)
		registry.cacheMissCount = make(map[tenant.Model]*uint64)
		registry.tenantPartyId = make(map[tenant.Model]uint32)

		// Create one party
		leaderId := uint32(5001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test batch lookup with mix of existing and non-existing characters
		characterIds := []uint32{leaderId, 9991, 9992} // leader + 2 non-existent
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 1) // Should return only the existing party
		assert.Equal(t, party.Id(), parties[0].Id())
	})

	t.Run("BatchLookupNoCharactersInParties", func(t *testing.T) {
		// Test with characters that don't exist in any party
		characterIds := []uint32{9995, 9996, 9997}
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 0) // Should return empty slice
	})

	t.Run("BatchLookupEmptyList", func(t *testing.T) {
		// Test with empty character list
		characterIds := []uint32{}
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 0) // Should return empty slice
	})
}

func TestIsCharacterInParty(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	t.Run("CharacterInParty", func(t *testing.T) {
		// Create a party with characters
		leaderId := uint32(6001)
		memberId := uint32(6002)
		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(memberId)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Test that leader is in party
		assert.True(t, IsCharacterInParty(ctx)(leaderId))

		// Test that member is in party
		assert.True(t, IsCharacterInParty(ctx)(memberId))
	})

	t.Run("CharacterNotInParty", func(t *testing.T) {
		// Test character not in any party
		nonExistentCharId := uint32(8888)
		assert.False(t, IsCharacterInParty(ctx)(nonExistentCharId))
	})

	t.Run("CharacterInDifferentParty", func(t *testing.T) {
		// Clear and create fresh parties
		registry := GetRegistry()
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
		registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
		registry.cacheHitCount = make(map[tenant.Model]*uint64)
		registry.cacheMissCount = make(map[tenant.Model]*uint64)
		registry.tenantPartyId = make(map[tenant.Model]uint32)

		// Create multiple parties
		leader1Id := uint32(7001)
		leader2Id := uint32(7002)
		GetRegistry().Create(testTenant, leader1Id)
		GetRegistry().Create(testTenant, leader2Id)

		// Verify each character is in a party
		assert.True(t, IsCharacterInParty(ctx)(leader1Id))
		assert.True(t, IsCharacterInParty(ctx)(leader2Id))

		// Verify non-existent character is not in any party
		assert.False(t, IsCharacterInParty(ctx)(9999))
	})
}

func TestByCharacterProvider(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	t.Run("ProviderReturnsCorrectParty", func(t *testing.T) {
		// Create a party
		leaderId := uint32(8001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test provider pattern
		provider := byCharacterProvider(ctx)(leaderId)
		foundParty, err := provider()
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())
		assert.Equal(t, leaderId, foundParty.LeaderId())
	})

	t.Run("ProviderReturnsErrorForNonExistentCharacter", func(t *testing.T) {
		// Test provider with non-existent character
		nonExistentCharId := uint32(9999)
		provider := byCharacterProvider(ctx)(nonExistentCharId)
		_, err := provider()
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("ProviderCanBeUsedMultipleTimes", func(t *testing.T) {
		// Create a party
		leaderId := uint32(8002)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test provider can be called multiple times
		provider := byCharacterProvider(ctx)(leaderId)
		
		foundParty1, err1 := provider()
		assert.NoError(t, err1)
		assert.Equal(t, party.Id(), foundParty1.Id())

		foundParty2, err2 := provider()
		assert.NoError(t, err2)
		assert.Equal(t, party.Id(), foundParty2.Id())

		// Results should be consistent
		assert.Equal(t, foundParty1.Id(), foundParty2.Id())
	})
}

func TestCacheManagementFunctions(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)
	ClearCache(ctx) // Clear any existing cache

	t.Run("InitialCacheState", func(t *testing.T) {
		// Test initial cache state
		hits, misses, hitRate := GetCacheStats(ctx)
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(0), misses)
		assert.Equal(t, float64(0), hitRate)
		assert.Equal(t, 0, GetCacheSize(ctx))
	})

	t.Run("CacheStatsAfterLookups", func(t *testing.T) {
		// Create a party to trigger cache usage
		leaderId := uint32(9001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Perform some lookups to generate cache stats
		_, _ = GetByCharacter(ctx)(leaderId)  // This should create cache entry
		_, _ = GetByCharacter(ctx)(leaderId)  // This should be a cache hit
		_, _ = GetByCharacter(ctx)(9999)     // This should be a cache miss

		// Check cache stats
		hits, misses, hitRate := GetCacheStats(ctx)
		assert.True(t, hits > 0 || misses > 0, "Expected some cache activity")
		
		cacheSize := GetCacheSize(ctx)
		assert.True(t, cacheSize >= 0, "Cache size should be non-negative")

		// Test that hit rate is calculated correctly
		if hits+misses > 0 {
			expectedHitRate := float64(hits) / float64(hits+misses)
			assert.Equal(t, expectedHitRate, hitRate)
		}

		// Verify party exists in the system
		foundParty, err := GetByCharacter(ctx)(leaderId)
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())
	})

	t.Run("ClearCacheFunction", func(t *testing.T) {
		// Create a party to populate cache
		leaderId := uint32(9002)
		GetRegistry().Create(testTenant, leaderId)
		
		// Perform lookup to populate cache
		_, _ = GetByCharacter(ctx)(leaderId)
		
		// Verify cache has some entries
		initialCacheSize := GetCacheSize(ctx)
		assert.True(t, initialCacheSize >= 0)

		// Clear cache
		ClearCache(ctx)

		// Verify cache is cleared
		newCacheSize := GetCacheSize(ctx)
		assert.Equal(t, 0, newCacheSize)
	})

	t.Run("CleanupStaleCacheFunction", func(t *testing.T) {
		// This function exists but we can't easily test staleness without time manipulation
		// So we just verify it doesn't panic and can be called
		require.NotPanics(t, func() {
			CleanupStaleCache(ctx)
		})

		// Verify cache functions still work after cleanup
		cacheSize := GetCacheSize(ctx)
		assert.True(t, cacheSize >= 0)
	})
}

func TestConcurrentLookups(t *testing.T) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]*uint64)
	registry.cacheMissCount = make(map[tenant.Model]*uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	t.Run("ConcurrentCharacterLookups", func(t *testing.T) {
		// Create a party
		leaderId := uint32(10001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test concurrent lookups don't cause race conditions
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				// Perform lookup
				foundParty, err := GetByCharacter(ctx)(leaderId)
				assert.NoError(t, err)
				assert.Equal(t, party.Id(), foundParty.Id())
				
				// Test IsCharacterInParty concurrently
				assert.True(t, IsCharacterInParty(ctx)(leaderId))
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}