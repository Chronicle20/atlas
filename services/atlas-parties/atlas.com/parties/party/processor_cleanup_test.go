package party

import (
	"atlas-parties/character"
	"atlas-parties/kafka/message"
	"atlas-parties/kafka/producer"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Additional edge case tests for character deletion handling scenarios
// These complement the main processor_test.go with specific deletion-related edge cases

func TestCharacterDeletion_EdgeCases(t *testing.T) {
	t.Run("registry error handling with invalid tenant", func(t *testing.T) {
		// Create invalid tenant
		invalidTenant := tenant.Model{}
		
		// These operations should handle gracefully
		require.NotPanics(t, func() {
			_, err := GetRegistry().Get(invalidTenant, 123)
			assert.Error(t, err)
		})
		
		require.NotPanics(t, func() {
			GetRegistry().Remove(invalidTenant, 123)
		})
	})
	
	t.Run("concurrent registry access", func(t *testing.T) {
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		party := GetRegistry().Create(ten, 123)
		
		// Simulate concurrent access
		go func() {
			GetRegistry().Update(ten, party.Id(), func(m Model) Model {
				return Model.AddMember(m, 456)
			})
		}()
		
		go func() {
			GetRegistry().Update(ten, party.Id(), func(m Model) Model {
				return Model.AddMember(m, 789)
			})
		}()
		
		// These should not cause data races or panics
		require.NotPanics(t, func() {
			updatedParty, _ := GetRegistry().Get(ten, party.Id())
			assert.NotNil(t, updatedParty)
		})
	})
}

func TestDeletionIdempotency(t *testing.T) {
	t.Run("multiple leave attempts for same character", func(t *testing.T) {
		// Use real processor setup from processor_test.go
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		
		ctx := tenant.WithContext(context.Background(), ten)
		processor := &ProcessorImpl{
			l:   logger,
			ctx: ctx,
			t:   ten,
			cp:  character.NewProcessor(logger, ctx),
			p:   nil,
		}
		
		// Create party and character
		party := GetRegistry().Create(ten, 123)
		party, _ = GetRegistry().Update(ten, party.Id(), func(m Model) Model {
			return Model.AddMember(m, 456)
		})
		
		// Create real character
		character.GetRegistry().Create(ten, 1, 1, 100000, 456, "TestChar", 50, 100, 0)
		character.GetRegistry().Update(ten, 456, func(m character.Model) character.Model {
			return m.JoinParty(party.Id())
		})
		
		buffer1 := message.NewBuffer()
		_, err1 := processor.Leave(buffer1)(party.Id(), 456)
		assert.NoError(t, err1)
		
		// Second leave attempt should fail gracefully
		buffer2 := message.NewBuffer()
		_, err2 := processor.Leave(buffer2)(party.Id(), 456)
		assert.Error(t, err2)
		assert.Equal(t, ErrNotIn, err2)
		
		// Cleanup
		character.GetRegistry().Delete(ten, 456)
	})
}

// Test for the specific character deletion event handling scenario
func TestCharacterDeletionEventFlow(t *testing.T) {
	t.Run("character deletion with party cleanup flow", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		
		// Create real processors
		ctx := tenant.WithContext(context.Background(), ten)
		charProcessor := character.NewProcessor(logger, ctx)
		partyProcessor := &ProcessorImpl{
			l:   logger,
			ctx: ctx,
			t:   ten,
			p:   producer.ProviderImpl(logger)(ctx), // Real producer that will fail to emit
			cp:  charProcessor,
			ip:  &mockInviteProcessor{}, // Add mock invite processor
		}
		
		// Create party with character
		party := GetRegistry().Create(ten, 123)
		
		// Create character in party
		character.GetRegistry().Create(ten, 1, 1, 100000, 456, "TestChar", 50, 100, 0)
		character.GetRegistry().Update(ten, 456, func(m character.Model) character.Model {
			return m.JoinParty(party.Id())
		})
		
		// Add character to party
		GetRegistry().Update(ten, party.Id(), func(m Model) Model {
			return Model.AddMember(m, 456)
		})
		
		// Verify setup
		_, err := character.GetRegistry().Get(ten, 456)
		require.NoError(t, err)
		currentParty, err := GetRegistry().Get(ten, party.Id())
		require.NoError(t, err)
		assert.Contains(t, currentParty.Members(), uint32(456))
		
		// Simulate the correct deletion flow: first party leave, then character deletion
		// This matches what handleStatusEventDeleted does in the character consumer
		
		// Step 1: Remove from party using party processor (like the event handler does)
		_, err = partyProcessor.LeaveAndEmit(party.Id(), 456)
		if err != nil {
			// Expect Kafka error in test environment, but party leave logic should work
			t.Logf("Expected Kafka emission error: %v", err)
		}
		
		// Step 2: Delete character from character registry (like the event handler does)
		err = charProcessor.Delete(456)
		assert.NoError(t, err)
		
		// Verify character is deleted
		_, err = character.GetRegistry().Get(ten, 456)
		assert.Error(t, err, "Character should be deleted")
		
		// Verify character is removed from party
		remainingParty, err := GetRegistry().Get(ten, party.Id())
		assert.NoError(t, err)
		assert.NotContains(t, remainingParty.Members(), uint32(456))
		
		// Cleanup
		GetRegistry().Remove(ten, party.Id())
	})

	// Additional character deletion scenarios
	t.Run("delete character not in party", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		
		ctx := tenant.WithContext(context.Background(), ten)
		charProcessor := character.NewProcessor(logger, ctx)
		
		// Create character not in any party
		character.GetRegistry().Create(ten, 1, 1, 100000, 123, "TestChar", 50, 100, 0)
		
		// Verify character exists
		_, err := character.GetRegistry().Get(ten, 123)
		require.NoError(t, err)
		
		// Delete character
		err = charProcessor.Delete(123)
		assert.NoError(t, err)
		
		// Verify character was removed from registry
		_, err = character.GetRegistry().Get(ten, 123)
		assert.Error(t, err, "Character should be deleted from registry")
	})

	t.Run("delete non-existent character is idempotent", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		
		ctx := tenant.WithContext(context.Background(), ten)
		charProcessor := character.NewProcessor(logger, ctx)
		
		// Try to delete character that doesn't exist
		err := charProcessor.Delete(999)
		
		// Should not return error (idempotent behavior)
		assert.NoError(t, err)
	})
	
	t.Run("delete party leader disbands party", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
		
		ctx := tenant.WithContext(context.Background(), ten)
		charProcessor := character.NewProcessor(logger, ctx)
		partyProcessor := &ProcessorImpl{
			l:   logger,
			ctx: ctx,
			t:   ten,
			p:   producer.ProviderImpl(logger)(ctx), // Real producer that will fail to emit
			cp:  charProcessor,
			ip:  &mockInviteProcessor{}, // Add mock invite processor
		}
		
		// Create party with leader
		leaderId := uint32(123)
		party := GetRegistry().Create(ten, leaderId)
		
		// Create leader character
		character.GetRegistry().Create(ten, 1, 1, 100000, leaderId, "Leader", 50, 100, 0)
		character.GetRegistry().Update(ten, leaderId, func(m character.Model) character.Model {
			return m.JoinParty(party.Id())
		})
		
		// Verify party exists
		_, err := GetRegistry().Get(ten, party.Id())
		require.NoError(t, err)
		
		// Simulate the correct deletion flow: first party leave (which disbands), then character deletion
		// This matches what handleStatusEventDeleted does in the character consumer
		
		// Step 1: Remove leader from party using party processor (this should disband the party)
		_, err = partyProcessor.LeaveAndEmit(party.Id(), leaderId)
		if err != nil {
			// Expect Kafka error in test environment, but party leave logic should work
			t.Logf("Expected Kafka emission error: %v", err)
		}
		
		// Step 2: Delete character from character registry
		err = charProcessor.Delete(leaderId)
		assert.NoError(t, err)
		
		// Verify character was removed from registry
		_, err = character.GetRegistry().Get(ten, leaderId)
		assert.Error(t, err, "Character should be deleted from registry")
		
		// Verify party was disbanded (should not exist)
		_, err = GetRegistry().Get(ten, party.Id())
		assert.Error(t, err, "Party should be disbanded when leader is deleted")
	})
}