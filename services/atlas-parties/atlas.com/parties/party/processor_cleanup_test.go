package party

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// Helper function to set up test environment
func setupTestEnvironment() (tenant.Model, context.Context, *logrus.Logger, *test.Hook) {
	// Setup test tenant and context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test and ensure proper initialization
	registry := GetRegistry()
	registry.partyReg = make(map[tenant.Model]map[uint32]Model)
	registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	registry.characterPartyCache = make(map[tenant.Model]map[uint32]*CacheEntry)
	registry.cacheHitCount = make(map[tenant.Model]uint64)
	registry.cacheMissCount = make(map[tenant.Model]uint64)
	registry.tenantPartyId = make(map[tenant.Model]uint32)

	// Use test logger to capture logs
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	return testTenant, ctx, logger, hook
}

func TestRemoveCharacterFromPartyCore(t *testing.T) {
	testTenant, ctx, logger, _ := setupTestEnvironment()

	t.Run("RemoveRegularMemberFromParty", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(1001)
		member1Id := uint32(1002)
		member2Id := uint32(1003)

		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(member1Id).AddMember(member2Id)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Verify initial state
		initialParty, err := GetRegistry().Get(testTenant, party.Id())
		assert.NoError(t, err)
		assert.Len(t, initialParty.Members(), 3)
		assert.Contains(t, initialParty.Members(), member1Id)

		// Test the core removal logic without event emission
		// Find the party containing the character
		foundParty, err := GetByCharacter(ctx)(member1Id)
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())

		// Remove the character from the party using registry directly
		updatedParty, err := GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { 
			return Model.RemoveMember(m, member1Id) 
		})
		assert.NoError(t, err)
		assert.NotContains(t, updatedParty.Members(), member1Id)
		assert.Contains(t, updatedParty.Members(), leaderId)
		assert.Contains(t, updatedParty.Members(), member2Id)
		assert.Len(t, updatedParty.Members(), 2)
		assert.Equal(t, leaderId, updatedParty.LeaderId()) // Leader should remain unchanged
	})

	t.Run("RemoveLeaderFromPartyTriggersElection", func(t *testing.T) {
		// Clear registry
		setupTestEnvironment()

		// Create a party with leader and members
		leaderId := uint32(2001)
		member1Id := uint32(2002)
		member2Id := uint32(2003)

		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(member1Id).AddMember(member2Id)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Remove the leader
		updatedParty, err := GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { 
			return Model.RemoveMember(m, leaderId) 
		})
		assert.NoError(t, err)
		assert.NotContains(t, updatedParty.Members(), leaderId)

		// Test leader election
		newLeaderId, err := electNewLeaderInternal(logger, ctx, updatedParty, leaderId)
		assert.NoError(t, err)
		assert.NotEqual(t, leaderId, newLeaderId)
		assert.Contains(t, []uint32{member1Id, member2Id}, newLeaderId)

		// Set the new leader
		finalParty, err := GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { 
			return Model.SetLeader(m, newLeaderId) 
		})
		assert.NoError(t, err)
		assert.Equal(t, newLeaderId, finalParty.LeaderId())
		assert.Contains(t, finalParty.Members(), newLeaderId)
	})

	t.Run("RemoveLastMemberDisbandParty", func(t *testing.T) {
		// Clear registry
		setupTestEnvironment()

		// Create a party with only a leader (single member)
		leaderId := uint32(3001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Verify party exists
		_, err := GetRegistry().Get(testTenant, party.Id())
		assert.NoError(t, err)

		// Remove the only member (disband the party)
		GetRegistry().Remove(testTenant, party.Id())

		// Verify party is removed
		_, err = GetRegistry().Get(testTenant, party.Id())
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestElectNewLeaderInternal(t *testing.T) {
	_, ctx, logger, _ := setupTestEnvironment()

	t.Run("ElectLeaderFromMultipleCandidates", func(t *testing.T) {
		// Create a party with multiple members
		leaderId := uint32(5001)
		member1Id := uint32(5002)
		member2Id := uint32(5003)
		member3Id := uint32(5004)

		party := Model{
			id:       12345,
			leaderId: leaderId,
			members:  []uint32{leaderId, member1Id, member2Id, member3Id},
		}

		// Elect new leader excluding the current leader
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, leaderId)

		assert.NoError(t, err)
		assert.NotEqual(t, leaderId, newLeaderId) // Should not elect the deleted leader
		assert.Contains(t, []uint32{member1Id, member2Id, member3Id}, newLeaderId) // Should elect one of the members
	})

	t.Run("ElectLeaderWithSingleCandidate", func(t *testing.T) {
		// Create a party with leader and one member
		leaderId := uint32(6001)
		memberId := uint32(6002)

		party := Model{
			id:       12346,
			leaderId: leaderId,
			members:  []uint32{leaderId, memberId},
		}

		// Elect new leader excluding the current leader
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, leaderId)

		assert.NoError(t, err)
		assert.Equal(t, memberId, newLeaderId) // Should elect the only remaining member
	})

	t.Run("ElectLeaderWithNoCandidates", func(t *testing.T) {
		// Create a party with only the leader
		leaderId := uint32(7001)

		party := Model{
			id:       12347,
			leaderId: leaderId,
			members:  []uint32{leaderId},
		}

		// Try to elect new leader excluding the only member
		_, err := electNewLeaderInternal(logger, ctx, party, leaderId)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no suitable candidate found for leader election")
	})

	t.Run("ElectLeaderExcludingDeletedCharacter", func(t *testing.T) {
		// Create a party where we're removing a non-leader member
		leaderId := uint32(8001)
		member1Id := uint32(8002)
		member2Id := uint32(8003)
		deletedMemberId := uint32(8004)

		party := Model{
			id:       12348,
			leaderId: leaderId,
			members:  []uint32{leaderId, member1Id, member2Id, deletedMemberId},
		}

		// Elect new leader excluding a specific member (not the leader)
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, deletedMemberId)

		assert.NoError(t, err)
		assert.NotEqual(t, deletedMemberId, newLeaderId) // Should not elect the deleted member
		assert.Contains(t, []uint32{leaderId, member1Id, member2Id}, newLeaderId) // Should elect one of the remaining members
	})
}

func TestPartyCleanupLogic(t *testing.T) {
	testTenant, ctx, _, _ := setupTestEnvironment()

	t.Run("PartyMembershipValidation", func(t *testing.T) {
		// Create a party
		leaderId := uint32(9001)
		memberId := uint32(9002)

		party := GetRegistry().Create(testTenant, leaderId)
		party = party.AddMember(memberId)
		GetRegistry().Update(testTenant, party.Id(), func(m Model) Model { return party })

		// Test membership validation
		assert.True(t, IsMember(party, leaderId))
		assert.True(t, IsMember(party, memberId))
		assert.False(t, IsMember(party, 99999)) // Non-existent member

		// Test leader validation
		assert.True(t, IsLeader(party, leaderId))
		assert.False(t, IsLeader(party, memberId))
	})

	t.Run("PartyIntegrityValidation", func(t *testing.T) {
		// Create a valid party
		party := Model{
			id:       10001,
			leaderId: 10001,
			members:  []uint32{10001, 10002, 10003},
		}

		// Valid party should pass integrity check
		err := ValidatePartyIntegrity(party)
		assert.NoError(t, err)

		// Invalid party (leader not in members) should fail
		invalidParty := Model{
			id:       10002,
			leaderId: 10010, // Leader not in members list
			members:  []uint32{10001, 10002, 10003},
		}

		err = ValidatePartyIntegrity(invalidParty)
		assert.Error(t, err)
	})

	t.Run("CharacterLookupInParty", func(t *testing.T) {
		// Create a party
		leaderId := uint32(11001)
		party := GetRegistry().Create(testTenant, leaderId)

		// Test character-to-party lookup
		foundParty, err := GetByCharacter(ctx)(leaderId)
		assert.NoError(t, err)
		assert.Equal(t, party.Id(), foundParty.Id())

		// Test lookup for non-existent character
		_, err = GetByCharacter(ctx)(99999)
		assert.Equal(t, ErrNotFound, err)

		// Test IsCharacterInParty
		assert.True(t, IsCharacterInParty(ctx)(leaderId))
		assert.False(t, IsCharacterInParty(ctx)(99999))
	})
}

func TestPartyCleanupEdgeCases(t *testing.T) {
	testTenant, ctx, _, _ := setupTestEnvironment()

	t.Run("RemoveNonExistentCharacter", func(t *testing.T) {
		// Try to find party for non-existent character
		_, err := GetByCharacter(ctx)(99999)
		assert.Equal(t, ErrNotFound, err)

		// This simulates the early return in RemoveCharacterFromPartyWithTransaction
		// when character is not found in any party
	})

	t.Run("ElectLeaderFromEmptyParty", func(t *testing.T) {
		// Create an empty party
		party := Model{
			id:       12349,
			leaderId: 0,
			members:  []uint32{},
		}

		// Create test logger
		logger, _ := test.NewNullLogger()

		// Try to elect leader from empty party
		_, err := electNewLeaderInternal(logger, ctx, party, 9001)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no members available for leader election")
	})

	t.Run("MultiPartyMembershipCheck", func(t *testing.T) {
		// Create multiple parties
		leader1Id := uint32(12001)
		leader2Id := uint32(12002)
		
		party1 := GetRegistry().Create(testTenant, leader1Id)
		party2 := GetRegistry().Create(testTenant, leader2Id)

		// Each character should be in their respective party
		foundParty1, err := GetByCharacter(ctx)(leader1Id)
		assert.NoError(t, err)
		assert.Equal(t, party1.Id(), foundParty1.Id())

		foundParty2, err := GetByCharacter(ctx)(leader2Id)
		assert.NoError(t, err)
		assert.Equal(t, party2.Id(), foundParty2.Id())

		// Parties should be different
		assert.NotEqual(t, party1.Id(), party2.Id())
	})
}