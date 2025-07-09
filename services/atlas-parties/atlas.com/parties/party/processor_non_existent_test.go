package party

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRemoveNonExistentCharacterFromParty(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Clear registry before test
	GetRegistry().partyReg = make(map[tenant.Model]map[uint32]Model)

	// Test removing a non-existent character
	nonExistentCharId := uint32(99999)
	transactionId := uuid.New()

	t.Run("RemoveNonExistentCharacterNoError", func(t *testing.T) {
		// This should not return an error and should handle gracefully
		result, err := RemoveCharacterFromPartyWithTransaction(logger, transactionId)(ctx)(nonExistentCharId)
		
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), result.Id()) // Should return empty model
	})
}

func TestLeaderElectionWithNonExistentCharacters(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Create a mock party with some non-existent characters
	party := Model{
		id:       12345,
		leaderId: 1001,
		members:  []uint32{1001, 1002, 1003}, // Assume these characters don't exist
	}

	t.Run("ElectLeaderWithAllNonExistentCharacters", func(t *testing.T) {
		// This should still elect a leader even if characters don't exist
		newLeaderId, err := electNewLeaderInternal(logger, ctx, party, 1001)
		
		assert.NoError(t, err)
		assert.True(t, newLeaderId == 1002 || newLeaderId == 1003) // Should elect one of the remaining members
		assert.NotEqual(t, uint32(1001), newLeaderId) // Should not elect the deleted character
	})
}

func TestEventEmissionWithNonExistentCharacters(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Create a mock party
	party := Model{
		id:       12345,
		leaderId: 1001,
		members:  []uint32{1001, 1002},
	}

	t.Run("EmitLeaderChangeEventWithNonExistentNewLeader", func(t *testing.T) {
		// This should not panic and should handle gracefully
		// We expect this to succeed with fallback world ID
		// The actual event emission might fail due to missing Kafka setup, but the function should handle non-existent characters gracefully
		// For this test, we just verify it doesn't panic and processes the non-existent character case
		assert.NotPanics(t, func() {
			_ = emitLeaderChangeEventInternal(logger, ctx, party, 1001, 1002)
		})
	})
}

func TestNonExistentCharacterValidationScenarios(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	t.Run("ValidateNonExistentCharacterMembership", func(t *testing.T) {
		// Test validating membership for non-existent character
		nonExistentCharId := uint32(99999)
		nonExistentPartyId := uint32(88888)
		
		err := ValidateMembership(ctx)(nonExistentPartyId, nonExistentCharId)
		
		// Should return ErrNotFound for non-existent party
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("CheckNonExistentCharacterInParty", func(t *testing.T) {
		// Test checking if non-existent character is in party
		nonExistentCharId := uint32(99999)
		
		result := IsCharacterInParty(ctx)(nonExistentCharId)
		
		// Should return false for non-existent character
		assert.False(t, result)
	})
}

func TestGracefulNonExistentCharacterHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Test various scenarios where characters might not exist
	t.Run("HandlesNonExistentCharacterGracefully", func(t *testing.T) {
		nonExistentCharId := uint32(99999)
		
		// Test GetByCharacter with non-existent character
		_, err := GetByCharacter(ctx)(nonExistentCharId)
		assert.Equal(t, ErrNotFound, err)
		
		// Test IsCharacterInParty with non-existent character
		result := IsCharacterInParty(ctx)(nonExistentCharId)
		assert.False(t, result)
		
		// Test GetPartiesByCharacters with non-existent characters
		parties, err := GetPartiesByCharacters(ctx)([]uint32{nonExistentCharId, 99998, 99997})
		assert.NoError(t, err)
		assert.Empty(t, parties) // Should return empty slice for non-existent characters
	})
}

func TestNonExistentCharacterDeletionScenarios(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Create a tenant and add it to context
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	t.Run("DeleteNonExistentCharacterFromParty", func(t *testing.T) {
		nonExistentCharId := uint32(99999)
		transactionId := uuid.New()
		
		// This should complete without error
		result, err := RemoveCharacterFromPartyWithTransaction(logger, transactionId)(ctx)(nonExistentCharId)
		
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), result.Id()) // Should return empty model
	})

	t.Run("ProcessNonExistentCharacterDeletionEvent", func(t *testing.T) {
		// Verify the system can handle deletion events for characters that don't exist
		// This is important for cases where character deletion events arrive after 
		// the character has already been cleaned up from the system
		
		nonExistentCharId := uint32(99999)
		
		// Test that looking up non-existent character returns appropriate error
		_, err := GetByCharacter(ctx)(nonExistentCharId)
		assert.Equal(t, ErrNotFound, err)
		
		// Test that character-to-party mapping handles non-existent characters
		result := IsCharacterInParty(ctx)(nonExistentCharId)
		assert.False(t, result)
		
		// These operations should complete successfully even for non-existent characters
		assert.NotPanics(t, func() {
			_, _ = GetByCharacter(ctx)(nonExistentCharId)
			_ = IsCharacterInParty(ctx)(nonExistentCharId)
		})
	})
}