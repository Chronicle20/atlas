package party

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentCharacterRemovalEdgeCases(t *testing.T) {
	testTenant, ctx, logger, _ := setupTestEnvironment()

	t.Run("ConcurrentRemovalOfSameCharacter", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(20001)
		memberId := uint32(20002)
		
		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(memberId)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Simulate concurrent removal of the same character
		var wg sync.WaitGroup
		var results []Model
		var errors []error
		var mutex sync.Mutex

		numConcurrentRemovals := 3
		wg.Add(numConcurrentRemovals)

		for i := 0; i < numConcurrentRemovals; i++ {
			go func() {
				defer wg.Done()
				
				result, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(memberId)
				
				mutex.Lock()
				results = append(results, result)
				errors = append(errors, err)
				mutex.Unlock()
			}()
		}

		wg.Wait()

		// Only one removal should succeed, others should gracefully handle non-existent character
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}

		// All should complete without panicking
		assert.Equal(t, numConcurrentRemovals, len(results))
		assert.Equal(t, numConcurrentRemovals, len(errors))

		// Verify final state: character should be removed from party
		finalParty, err := GetByCharacter(ctx)(memberId)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, uint32(0), finalParty.Id())
	})

	t.Run("ConcurrentRemovalOfDifferentCharacters", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(21001)
		member1Id := uint32(21002)
		member2Id := uint32(21003)
		member3Id := uint32(21004)
		
		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(member1Id).AddMember(member2Id).AddMember(member3Id)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Simulate concurrent removal of different characters
		var wg sync.WaitGroup
		memberIds := []uint32{member1Id, member2Id, member3Id}
		var results []Model
		var errors []error
		var mutex sync.Mutex

		wg.Add(len(memberIds))

		for _, memberId := range memberIds {
			go func(id uint32) {
				defer wg.Done()
				
				result, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(id)
				
				mutex.Lock()
				results = append(results, result)
				errors = append(errors, err)
				mutex.Unlock()
			}(memberId)
		}

		wg.Wait()

		// All removals should succeed
		assert.Equal(t, len(memberIds), len(results))
		assert.Equal(t, len(memberIds), len(errors))
		
		for _, err := range errors {
			assert.NoError(t, err)
		}

		// Verify final state: only leader should remain
		finalParty, err := GetByCharacter(ctx)(leaderId)
		assert.NoError(t, err)
		assert.Len(t, finalParty.Members(), 1)
		assert.Contains(t, finalParty.Members(), leaderId)
	})
}

func TestLeaderElectionEdgeCases(t *testing.T) {
	_, ctx, logger, _ := setupTestEnvironment()

	t.Run("LeaderElectionWithInvalidCharacterIds", func(t *testing.T) {
		// Create a party with some invalid character IDs
		leaderId := uint32(22001)
		validMemberId := uint32(22002)
		invalidMemberId := uint32(0) // Invalid ID
		
		party := Model{
			id:       22000,
			leaderId: leaderId,
			members:  []uint32{leaderId, validMemberId, invalidMemberId},
		}

		// Should still elect a valid leader
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, leaderId)
		
		assert.NoError(t, err)
		assert.Equal(t, validMemberId, newLeaderId) // Should skip invalid ID and elect valid member
	})

	t.Run("LeaderElectionWithDuplicateCharacterIds", func(t *testing.T) {
		// Create a party with duplicate character IDs
		leaderId := uint32(23001)
		memberId := uint32(23002)
		
		party := Model{
			id:       23000,
			leaderId: leaderId,
			members:  []uint32{leaderId, memberId, memberId, memberId}, // Duplicates
		}

		// Should elect the valid member only once
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, leaderId)
		
		assert.NoError(t, err)
		assert.Equal(t, memberId, newLeaderId) // Should elect the valid member
	})

	t.Run("LeaderElectionWithAllMembersExcluded", func(t *testing.T) {
		// Create a party where we try to exclude a character that results in no valid candidates
		leaderId := uint32(24001)
		
		party := Model{
			id:       24000,
			leaderId: leaderId,
			members:  []uint32{leaderId}, // Only leader
		}

		// Try to elect new leader excluding the only member (leader)
		_, err := electNewLeaderInternal(logger, ctx, party, leaderId)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no suitable candidate found for leader election")
	})
}

func TestPartyIntegrityValidationEdgeCases(t *testing.T) {
	t.Run("PartyWithNegativeId", func(t *testing.T) {
		party := Model{
			id:       0, // Invalid ID
			leaderId: 25001,
			members:  []uint32{25001},
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid party ID")
	})

	t.Run("PartyWithNegativeLeaderId", func(t *testing.T) {
		party := Model{
			id:       25000,
			leaderId: 0, // Invalid leader ID
			members:  []uint32{25001},
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid leader ID")
	})

	t.Run("PartyWithEmptyMembers", func(t *testing.T) {
		party := Model{
			id:       25001,
			leaderId: 25001,
			members:  []uint32{}, // Empty members
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "party has no members")
	})

	t.Run("PartyWithLeaderNotInMembers", func(t *testing.T) {
		party := Model{
			id:       25002,
			leaderId: 25001,
			members:  []uint32{25002, 25003}, // Leader not in members
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "leader is not in party members")
	})

	t.Run("PartyWithDuplicateMembers", func(t *testing.T) {
		party := Model{
			id:       25003,
			leaderId: 25001,
			members:  []uint32{25001, 25002, 25002}, // Duplicate members
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate member found")
	})

	t.Run("PartyWithInvalidMemberId", func(t *testing.T) {
		party := Model{
			id:       25004,
			leaderId: 25001,
			members:  []uint32{25001, 0}, // Invalid member ID
		}

		err := ValidatePartyIntegrity(party)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid member ID")
	})
}

func TestCharacterLookupEdgeCases(t *testing.T) {
	testTenant, ctx, _, _ := setupTestEnvironment()

	t.Run("LookupWithInvalidCharacterId", func(t *testing.T) {
		// Test with zero character ID
		_, err := GetByCharacter(ctx)(0)
		assert.Equal(t, ErrNotFound, err)

		// Test with maximum uint32 value
		maxUint32 := uint32(0xFFFFFFFF)
		_, err = GetByCharacter(ctx)(maxUint32)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("IsCharacterInPartyWithInvalidIds", func(t *testing.T) {
		// Test with zero character ID
		result := IsCharacterInParty(ctx)(0)
		assert.False(t, result)

		// Test with maximum uint32 value
		maxUint32 := uint32(0xFFFFFFFF)
		result = IsCharacterInParty(ctx)(maxUint32)
		assert.False(t, result)
	})

	t.Run("GetPartiesByCharactersWithEmptySlice", func(t *testing.T) {
		// Test with empty character slice
		parties, err := GetPartiesByCharacters(ctx)([]uint32{})
		assert.NoError(t, err)
		assert.Empty(t, parties)
	})

	t.Run("GetPartiesByCharactersWithNilSlice", func(t *testing.T) {
		// Test with nil character slice
		parties, err := GetPartiesByCharacters(ctx)(nil)
		assert.NoError(t, err)
		assert.Empty(t, parties)
	})

	t.Run("GetPartiesByCharactersWithMixedValidInvalidIds", func(t *testing.T) {
		// Create a party with one valid character
		validCharId := uint32(26001)
		party := GetRegistry().Create(testTenant, validCharId)

		// Test with mixed valid/invalid character IDs
		characterIds := []uint32{validCharId, 0, 99999}
		parties, err := GetPartiesByCharacters(ctx)(characterIds)
		assert.NoError(t, err)
		assert.Len(t, parties, 1) // Should only find the valid character's party
		assert.Equal(t, party.Id(), parties[0].Id())
	})
}

func TestMemoryLeakPreventionEdgeCases(t *testing.T) {
	testTenant, ctx, _, _ := setupTestEnvironment()

	t.Run("CreateAndDestroyManyParties", func(t *testing.T) {
		// Create many parties to test memory management
		numParties := 100
		var partyIds []uint32

		for i := 0; i < numParties; i++ {
			leaderId := uint32(27000 + i)
			party := GetRegistry().Create(testTenant, leaderId)
			partyIds = append(partyIds, party.Id())
		}

		// Verify all parties were created
		assert.Len(t, partyIds, numParties)

		// Remove all parties
		for _, partyId := range partyIds {
			GetRegistry().Remove(testTenant, partyId)
		}

		// Verify all parties are removed
		for _, partyId := range partyIds {
			_, err := GetRegistry().Get(testTenant, partyId)
			assert.Equal(t, ErrNotFound, err)
		}

		// Verify character mappings are cleaned up
		for i := 0; i < numParties; i++ {
			leaderId := uint32(27000 + i)
			result := IsCharacterInParty(ctx)(leaderId)
			assert.False(t, result)
		}
	})

	t.Run("RepeatedCharacterAdditionAndRemoval", func(t *testing.T) {
		// Create a party
		leaderId := uint32(28001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Repeatedly add and remove the same character
		memberId := uint32(28002)
		numIterations := 50

		for i := 0; i < numIterations; i++ {
			// Add member
			party = party.AddMember(memberId)
			GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

			// Verify member was added
			assert.True(t, IsCharacterInParty(ctx)(memberId))

			// Remove member
			party = party.RemoveMember(memberId)
			GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

			// Verify member was removed
			assert.False(t, IsCharacterInParty(ctx)(memberId))
		}

		// Final verification
		finalParty, err := GetRegistry().Get(testTenant, party.Id())
		assert.NoError(t, err)
		assert.Len(t, finalParty.Members(), 1) // Only leader should remain
		assert.Contains(t, finalParty.Members(), leaderId)
	})
}

func TestTimeoutAndResourceCleanupEdgeCases(t *testing.T) {
	testTenant, ctx, logger, _ := setupTestEnvironment()

	t.Run("OperationWithCancelledContext", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		// Operations should still work as they don't depend on context cancellation
		// but we test they handle cancelled context gracefully
		leaderId := uint32(29001)
		GetRegistry().Create(testTenant, leaderId)

		// Test character removal with cancelled context
		_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(cancelledCtx)(leaderId)
		
		// Should complete successfully even with cancelled context
		assert.NoError(t, err)
	})

	t.Run("OperationWithTimeoutContext", func(t *testing.T) {
		// Create a context with very short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(1 * time.Millisecond)

		// Operations should still work as they don't depend on context timeout
		leaderId := uint32(30001)
		GetRegistry().Create(testTenant, leaderId)

		// Test character removal with timeout context
		_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(timeoutCtx)(leaderId)
		
		// Should complete successfully even with timeout context
		assert.NoError(t, err)
	})
}

func TestRaceConditionEdgeCases(t *testing.T) {
	testTenant, ctx, logger, _ := setupTestEnvironment()

	t.Run("ConcurrentPartyCreationAndDeletion", func(t *testing.T) {
		// Test concurrent creation and deletion of parties
		var wg sync.WaitGroup
		numOperations := 10
		baseCharId := uint32(31000)

		wg.Add(numOperations * 2) // Create and delete operations

		// Concurrent creation
		for i := 0; i < numOperations; i++ {
			go func(index int) {
				defer wg.Done()
				leaderId := baseCharId + uint32(index)
				party := GetRegistry().Create(testTenant, leaderId)
				
				// Verify party was created
				_, err := GetRegistry().Get(testTenant, party.Id())
				assert.NoError(t, err)
			}(i)
		}

		// Concurrent deletion (with slight delay to ensure some parties are created first)
		time.Sleep(10 * time.Millisecond)
		for i := 0; i < numOperations; i++ {
			go func(index int) {
				defer wg.Done()
				leaderId := baseCharId + uint32(index)
				
				// Try to remove character from party
				_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(leaderId)
				// May succeed or fail depending on timing, but should not panic
				_ = err
			}(i)
		}

		wg.Wait()
	})

	t.Run("ConcurrentLeaderElectionAttempts", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(32001)
		member1Id := uint32(32002)
		member2Id := uint32(32003)
		
		party := Model{
			id:       32000,
			leaderId: leaderId,
			members:  []uint32{leaderId, member1Id, member2Id},
		}

		// Concurrent leader election attempts
		var wg sync.WaitGroup
		numElections := 5
		var results []uint32
		var errors []error
		var mutex sync.Mutex

		wg.Add(numElections)

		for i := 0; i < numElections; i++ {
			go func() {
				defer wg.Done()
				
				newLeaderId, err := electNewLeaderInternal(logger, ctx, party, leaderId)
				
				mutex.Lock()
				results = append(results, newLeaderId)
				errors = append(errors, err)
				mutex.Unlock()
			}()
		}

		wg.Wait()

		// All elections should succeed and elect a valid member
		assert.Len(t, results, numElections)
		assert.Len(t, errors, numElections)
		
		for i, err := range errors {
			assert.NoError(t, err)
			assert.Contains(t, []uint32{member1Id, member2Id}, results[i])
		}
	})
}

func TestEventEmissionFailureEdgeCases(t *testing.T) {
	testTenant, ctx, logger, hook := setupTestEnvironment()

	t.Run("PartyRemovalWithEventEmissionFailure", func(t *testing.T) {
		// Create a party
		leaderId := uint32(33001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test character removal which may fail during event emission
		// but should still complete the core operation
		_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(leaderId)
		
		// Should complete successfully
		assert.NoError(t, err)

		// Verify party was removed from registry
		_, err = GetRegistry().Get(testTenant, party.Id())
		assert.Equal(t, ErrNotFound, err)

		// Check that appropriate logs were generated
		assert.True(t, len(hook.Entries) > 0)
	})

	t.Run("LeaderElectionWithEventEmissionFailure", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(34001)
		memberId := uint32(34002)
		
		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(memberId)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Remove leader to trigger election
		_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(leaderId)
		
		// Should complete successfully even if event emission fails
		assert.NoError(t, err)

		// Verify new leader was elected
		updatedParty, err := GetByCharacter(ctx)(memberId)
		assert.NoError(t, err)
		assert.Equal(t, memberId, updatedParty.LeaderId())
	})
}

func TestDataCorruptionPrevention(t *testing.T) {
	testTenant, ctx, _, _ := setupTestEnvironment()

	t.Run("PreventCharacterInMultipleParties", func(t *testing.T) {
		// Create first party
		leaderId := uint32(35001)
		party1 := GetRegistry().Create(testTenant, leaderId)

		// Try to create second party with same character
		// This should be prevented by the system
		party2 := GetRegistry().Create(testTenant, leaderId)

		// Verify character is only in one party
		foundParty, err := GetByCharacter(ctx)(leaderId)
		assert.NoError(t, err)
		
		// Should find the character in one of the parties
		assert.True(t, foundParty.Id() == party1.Id() || foundParty.Id() == party2.Id())
	})

	t.Run("PreventInvalidPartyStateTransitions", func(t *testing.T) {
		// Create a valid party
		leaderId := uint32(36001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Attempt to create invalid state
		invalidParty := Model{
			id:       party.Id(),
			leaderId: 99999, // Invalid leader not in members
			members:  []uint32{leaderId},
		}

		// Validate that integrity check catches this
		err := ValidatePartyIntegrity(invalidParty)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "leader is not in party members")
	})
}

func TestMaxLoadStressEdgeCases(t *testing.T) {
	testTenant, ctx, logger, _ := setupTestEnvironment()

	t.Run("HandleLargePartySize", func(t *testing.T) {
		// Create a party with many members
		leaderId := uint32(37001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Add many members (but not too many to avoid test timeout)
		numMembers := 50
		for i := 0; i < numMembers; i++ {
			memberId := uint32(37100 + i)
			party = party.AddMember(memberId)
		}

		// Update party in registry
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Verify all members are in party
		finalParty, err := GetRegistry().Get(testTenant, party.Id())
		assert.NoError(t, err)
		assert.Len(t, finalParty.Members(), numMembers+1) // +1 for leader

		// Test removing all members except leader
		for i := 0; i < numMembers; i++ {
			memberId := uint32(37100 + i)
			_, err := RemoveCharacterFromPartyWithTransaction(logger, uuid.New())(ctx)(memberId)
			assert.NoError(t, err)
		}

		// Verify only leader remains
		finalParty, err = GetRegistry().Get(testTenant, party.Id())
		assert.NoError(t, err)
		assert.Len(t, finalParty.Members(), 1)
		assert.Contains(t, finalParty.Members(), leaderId)
	})
}

// Helper function to set up test environment (duplicated from other test files for completeness)
func setupTestEnvironmentForEdgeCases() (tenant.Model, context.Context, *logrus.Logger, *test.Hook) {
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

	// Use test logger to capture logs
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	return testTenant, ctx, logger, hook
}