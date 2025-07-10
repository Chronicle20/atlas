package character

import (
	"atlas-parties/party"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCharacterDeletionIntegration tests the complete end-to-end flow
// from receiving a character deletion event through party cleanup
func TestCharacterDeletionIntegration(t *testing.T) {
	// Setup test environment
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Reset idempotency tracker for clean test state
	originalTracker := deletionIdempotencyTracker
	defer func() {
		deletionIdempotencyTracker = originalTracker
	}()

	deletionIdempotencyTracker = &idempotencyTracker{
		processedEvents: make(map[uuid.UUID]*processedEvent),
		lastCleanup:     time.Now(),
		cleanupInterval: 5 * time.Minute,
		eventTTL:        30 * time.Minute,
	}

	// Clear party registry for clean test state
	registry := party.GetRegistry()
	registry.ClearCache(testTenant)

	t.Run("CharacterDeletionWithPartyLeaderElection", func(t *testing.T) {
		hook.Reset()
		
		// Setup: Create a party with multiple members
		leaderId := uint32(1001)
		member1Id := uint32(1002)
		member2Id := uint32(1003)

		// Create party with leader
		testParty := party.GetRegistry().Create(testTenant, leaderId)
		testParty = testParty.AddMember(member1Id).AddMember(member2Id)
		party.GetRegistry().Update(testTenant, testParty.Id(), func(m party.Model) party.Model { 
			return testParty 
		})

		// Verify initial party state
		initialParty, err := party.GetByCharacter(ctx)(leaderId)
		require.NoError(t, err)
		assert.Equal(t, leaderId, initialParty.LeaderId())
		assert.Len(t, initialParty.Members(), 3)
		assert.Contains(t, initialParty.Members(), leaderId)
		assert.Contains(t, initialParty.Members(), member1Id)
		assert.Contains(t, initialParty.Members(), member2Id)

		// Create deletion event for the leader
		transactionId := uuid.New()
		deletionEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   leaderId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Process deletion event
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, deletionEvent)
		})

		// Verify leader was removed and new leader elected
		updatedParty, err := party.GetByCharacter(ctx)(member1Id)
		require.NoError(t, err)
		assert.NotEqual(t, leaderId, updatedParty.LeaderId(), "Original leader should be removed")
		assert.Contains(t, []uint32{member1Id, member2Id}, updatedParty.LeaderId(), "New leader should be one of remaining members")
		assert.Len(t, updatedParty.Members(), 2, "Party should have 2 members after leader removal")
		assert.NotContains(t, updatedParty.Members(), leaderId, "Deleted leader should not be in party")

		// Verify event was marked as processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, leaderId))

		// Verify logging occurred
		logEntries := hook.AllEntries()
		assert.True(t, len(logEntries) > 0, "Should have logged the deletion process")

		// Check for specific log messages indicating successful processing
		foundProcessingLog := false
		foundLeaderElectionLog := false
		for _, entry := range logEntries {
			if entry.Message == "Processing character deletion event for character [1001]." {
				foundProcessingLog = true
			}
			if entry.Data["isLeader"] == true {
				foundLeaderElectionLog = true
			}
		}
		assert.True(t, foundProcessingLog, "Should log that deletion event is being processed")
		assert.True(t, foundLeaderElectionLog, "Should log that deleted character was a leader")
	})

	t.Run("CharacterDeletionWithPartyDisbanding", func(t *testing.T) {
		hook.Reset()
		
		// Setup: Create a party with only one member (leader)
		soloLeaderId := uint32(2001)
		_ = party.GetRegistry().Create(testTenant, soloLeaderId)

		// Verify party exists
		initialParty, err := party.GetByCharacter(ctx)(soloLeaderId)
		require.NoError(t, err)
		assert.Equal(t, soloLeaderId, initialParty.LeaderId())
		assert.Len(t, initialParty.Members(), 1)

		// Create deletion event for the solo leader
		transactionId := uuid.New()
		deletionEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   soloLeaderId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Process deletion event
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, deletionEvent)
		})

		// Verify party was disbanded (character not found in any party)
		_, err = party.GetByCharacter(ctx)(soloLeaderId)
		assert.Equal(t, party.ErrNotFound, err, "Party should be disbanded after last member deletion")

		// Verify event was marked as processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, soloLeaderId))

		// Verify appropriate logging
		logEntries := hook.AllEntries()
		assert.True(t, len(logEntries) > 0, "Should have logged the deletion process")
	})

	t.Run("CharacterDeletionWithNonPartyMember", func(t *testing.T) {
		hook.Reset()
		
		// Setup: Character not in any party
		nonPartyCharacterId := uint32(3001)

		// Verify character is not in any party
		_, err := party.GetByCharacter(ctx)(nonPartyCharacterId)
		assert.Equal(t, party.ErrNotFound, err, "Character should not be in any party initially")

		// Create deletion event for non-party character
		transactionId := uuid.New()
		deletionEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   nonPartyCharacterId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Process deletion event
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, deletionEvent)
		})

		// Verify event was still marked as processed (idempotency)
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, nonPartyCharacterId))

		// Verify appropriate logging for non-party member
		logEntries := hook.AllEntries()
		foundNonPartyLog := false
		for _, entry := range logEntries {
			if entry.Message == "Character [3001] not found in any party before deletion." {
				foundNonPartyLog = true
			}
		}
		assert.True(t, foundNonPartyLog, "Should log when character is not in any party")
	})

	t.Run("MultipleDeletionEventsIdempotency", func(t *testing.T) {
		hook.Reset()
		
		// Setup: Create a party
		leaderId := uint32(4001)
		memberId := uint32(4002)
		testParty := party.GetRegistry().Create(testTenant, leaderId)
		testParty = testParty.AddMember(memberId)
		party.GetRegistry().Update(testTenant, testParty.Id(), func(m party.Model) party.Model { 
			return testParty 
		})

		// Create deletion event
		transactionId := uuid.New()
		deletionEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   leaderId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Process deletion event first time
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, deletionEvent)
		})

		// Verify processing occurred
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, leaderId))
		initialEventCount := deletionIdempotencyTracker.getProcessedEventCount()

		// Get party state after first processing
		updatedParty, err := party.GetByCharacter(ctx)(memberId)
		require.NoError(t, err)
		memberCountAfterFirst := len(updatedParty.Members())

		// Process the same deletion event again (duplicate)
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, deletionEvent)
		})

		// Verify idempotency: event count should not increase
		assert.Equal(t, initialEventCount, deletionIdempotencyTracker.getProcessedEventCount(), "Duplicate event should not increase processed count")

		// Verify party state unchanged by duplicate processing
		finalParty, err := party.GetByCharacter(ctx)(memberId)
		require.NoError(t, err)
		assert.Equal(t, memberCountAfterFirst, len(finalParty.Members()), "Party state should not change on duplicate processing")

		// Verify appropriate logging for duplicate detection
		logEntries := hook.AllEntries()
		foundDuplicateLog := false
		for _, entry := range logEntries {
			if entry.Message == "Character deletion event for character [4001] already processed, skipping duplicate." {
				foundDuplicateLog = true
			}
		}
		assert.True(t, foundDuplicateLog, "Should log when duplicate event is detected")
	})

	t.Run("MalformedEventHandling", func(t *testing.T) {
		hook.Reset()
		
		// Test various malformed events
		malformedEvents := []struct {
			name  string
			event StatusEvent[StatusEventDeletedBody]
		}{
			{
				name: "InvalidCharacterId",
				event: StatusEvent[StatusEventDeletedBody]{
					TransactionId: uuid.New(),
					WorldId:       1,
					CharacterId:   0, // Invalid
					Type:          StatusEventTypeDeleted,
					Body:          StatusEventDeletedBody{},
				},
			},
			{
				name: "InvalidWorldId",
				event: StatusEvent[StatusEventDeletedBody]{
					TransactionId: uuid.New(),
					WorldId:       0, // Invalid
					CharacterId:   5001,
					Type:          StatusEventTypeDeleted,
					Body:          StatusEventDeletedBody{},
				},
			},
			{
				name: "WrongEventType",
				event: StatusEvent[StatusEventDeletedBody]{
					TransactionId: uuid.New(),
					WorldId:       1,
					CharacterId:   5002,
					Type:          StatusEventTypeLogin, // Wrong type
					Body:          StatusEventDeletedBody{},
				},
			},
		}

		for _, tc := range malformedEvents {
			t.Run(tc.name, func(t *testing.T) {
				initialEventCount := deletionIdempotencyTracker.getProcessedEventCount()

				// Process malformed event
				require.NotPanics(t, func() {
					handleStatusEventDeleted(logger, ctx, tc.event)
				})

				// Verify malformed events are not processed
				if tc.name == "WrongEventType" {
					// Wrong event type should be ignored completely
					assert.Equal(t, initialEventCount, deletionIdempotencyTracker.getProcessedEventCount(), "Wrong event type should not be processed")
					assert.False(t, deletionIdempotencyTracker.isEventProcessed(tc.event.TransactionId, tc.event.CharacterId))
				} else {
					// Validation failures should also not be processed
					assert.Equal(t, initialEventCount, deletionIdempotencyTracker.getProcessedEventCount(), "Invalid events should not be processed")
					assert.False(t, deletionIdempotencyTracker.isEventProcessed(tc.event.TransactionId, tc.event.CharacterId))
				}
			})
		}
	})
}

// TestConcurrentCharacterDeletions tests handling of concurrent deletion events
func TestConcurrentCharacterDeletions(t *testing.T) {
	// Setup test environment
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Reset idempotency tracker
	originalTracker := deletionIdempotencyTracker
	defer func() {
		deletionIdempotencyTracker = originalTracker
	}()

	deletionIdempotencyTracker = &idempotencyTracker{
		processedEvents: make(map[uuid.UUID]*processedEvent),
		lastCleanup:     time.Now(),
		cleanupInterval: 5 * time.Minute,
		eventTTL:        30 * time.Minute,
	}

	// Clear party registry
	registry := party.GetRegistry()
	registry.ClearCache(testTenant)

	t.Run("ConcurrentDeletionsInSameParty", func(t *testing.T) {
		// Setup: Create a party with multiple members
		leaderId := uint32(6001)
		member1Id := uint32(6002)
		member2Id := uint32(6003)
		member3Id := uint32(6004)

		testParty := party.GetRegistry().Create(testTenant, leaderId)
		testParty = testParty.AddMember(member1Id).AddMember(member2Id).AddMember(member3Id)
		party.GetRegistry().Update(testTenant, testParty.Id(), func(m party.Model) party.Model { 
			return testParty 
		})

		// Create deletion events for multiple members
		deletionEvents := []StatusEvent[StatusEventDeletedBody]{
			{
				TransactionId: uuid.New(),
				WorldId:       1,
				CharacterId:   member1Id,
				Type:          StatusEventTypeDeleted,
				Body:          StatusEventDeletedBody{},
			},
			{
				TransactionId: uuid.New(),
				WorldId:       1,
				CharacterId:   member2Id,
				Type:          StatusEventTypeDeleted,
				Body:          StatusEventDeletedBody{},
			},
		}

		// Process deletion events concurrently
		var wg sync.WaitGroup
		wg.Add(len(deletionEvents))

		for _, event := range deletionEvents {
			go func(e StatusEvent[StatusEventDeletedBody]) {
				defer wg.Done()
				require.NotPanics(t, func() {
					handleStatusEventDeleted(logger, ctx, e)
				})
			}(event)
		}

		wg.Wait()

		// Verify both events were processed
		for _, event := range deletionEvents {
			assert.True(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
		}

		// Verify remaining party state is consistent
		remainingParty, err := party.GetByCharacter(ctx)(leaderId)
		require.NoError(t, err)
		assert.Contains(t, remainingParty.Members(), leaderId, "Leader should still be in party")
		assert.Contains(t, remainingParty.Members(), member3Id, "Non-deleted member should still be in party")
		assert.NotContains(t, remainingParty.Members(), member1Id, "Deleted member should not be in party")
		assert.NotContains(t, remainingParty.Members(), member2Id, "Deleted member should not be in party")
		assert.Equal(t, 2, len(remainingParty.Members()), "Party should have 2 remaining members")
	})

	t.Run("ConcurrentIdenticalDeletions", func(t *testing.T) {
		// Setup: Create a party
		leaderId := uint32(7001)
		memberId := uint32(7002)
		testParty := party.GetRegistry().Create(testTenant, leaderId)
		testParty = testParty.AddMember(memberId)
		party.GetRegistry().Update(testTenant, testParty.Id(), func(m party.Model) party.Model { 
			return testParty 
		})

		// Create identical deletion events
		transactionId := uuid.New()
		deletionEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   leaderId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Process the same event concurrently multiple times
		var wg sync.WaitGroup
		concurrentRequests := 5
		wg.Add(concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			go func() {
				defer wg.Done()
				require.NotPanics(t, func() {
					handleStatusEventDeleted(logger, ctx, deletionEvent)
				})
			}()
		}

		wg.Wait()

		// Verify event was only processed once due to idempotency
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, leaderId))
		
		// Verify party state is consistent
		finalParty, err := party.GetByCharacter(ctx)(memberId)
		require.NoError(t, err)
		assert.Equal(t, memberId, finalParty.LeaderId(), "Remaining member should be new leader")
		assert.Len(t, finalParty.Members(), 1, "Party should have 1 remaining member")
		assert.NotContains(t, finalParty.Members(), leaderId, "Deleted leader should not be in party")
	})
}

// TestKafkaConsumerErrorRecovery tests the error recovery mechanisms
func TestKafkaConsumerErrorRecovery(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Reset idempotency tracker
	originalTracker := deletionIdempotencyTracker
	defer func() {
		deletionIdempotencyTracker = originalTracker
	}()

	deletionIdempotencyTracker = &idempotencyTracker{
		processedEvents: make(map[uuid.UUID]*processedEvent),
		lastCleanup:     time.Now(),
		cleanupInterval: 5 * time.Minute,
		eventTTL:        30 * time.Minute,
	}

	t.Run("HandlerRemainsStableAfterErrors", func(t *testing.T) {
		hook.Reset()
		
		// Process a series of both valid and invalid events
		events := []StatusEvent[StatusEventDeletedBody]{
			// Valid event
			{
				TransactionId: uuid.New(),
				WorldId:       1,
				CharacterId:   8001,
				Type:          StatusEventTypeDeleted,
				Body:          StatusEventDeletedBody{},
			},
			// Invalid event (bad world ID)
			{
				TransactionId: uuid.New(),
				WorldId:       0, // Invalid
				CharacterId:   8002,
				Type:          StatusEventTypeDeleted,
				Body:          StatusEventDeletedBody{},
			},
			// Another valid event
			{
				TransactionId: uuid.New(),
				WorldId:       1,
				CharacterId:   8003,
				Type:          StatusEventTypeDeleted,
				Body:          StatusEventDeletedBody{},
			},
		}

		// Process all events - should not panic on any
		for _, event := range events {
			require.NotPanics(t, func() {
				handleStatusEventDeleted(logger, ctx, event)
			}, "Handler should not panic on any event type")
		}

		// Verify only valid events were processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(events[0].TransactionId, events[0].CharacterId))
		assert.False(t, deletionIdempotencyTracker.isEventProcessed(events[1].TransactionId, events[1].CharacterId))
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(events[2].TransactionId, events[2].CharacterId))

		// Verify appropriate error logging occurred
		logEntries := hook.AllEntries()
		foundErrorLog := false
		for _, entry := range logEntries {
			if entry.Level == logrus.ErrorLevel && entry.Message == "Malformed character deletion event received, skipping processing." {
				foundErrorLog = true
			}
		}
		assert.True(t, foundErrorLog, "Should log errors for malformed events")
	})
}

// TestIdempotencyTrackerIntegration tests the idempotency tracker in integration scenarios
func TestIdempotencyTrackerIntegration(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	t.Run("IdempotencyTrackerCleanupDuringProcessing", func(t *testing.T) {
		// Create tracker with very short cleanup interval for testing
		testTracker := &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now().Add(-10 * time.Minute), // Force cleanup
			cleanupInterval: 1 * time.Millisecond, // Very short
			eventTTL:        5 * time.Millisecond,  // Very short
		}

		// Temporarily replace global tracker
		originalTracker := deletionIdempotencyTracker
		deletionIdempotencyTracker = testTracker
		defer func() {
			deletionIdempotencyTracker = originalTracker
		}()

		// Add some old events manually
		oldTxId := uuid.New()
		testTracker.processedEvents[oldTxId] = &processedEvent{
			transactionId: oldTxId,
			characterId:   9001,
			processedAt:   time.Now().Add(-10 * time.Millisecond), // Older than TTL
		}

		assert.Equal(t, 1, testTracker.getProcessedEventCount())

		// Process a new event which should trigger cleanup
		newEvent := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   9002,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Wait a bit to ensure old event expires
		time.Sleep(10 * time.Millisecond)

		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, newEvent)
		})

		// Verify cleanup occurred - old event should be removed, new event should be added
		assert.False(t, testTracker.isEventProcessed(oldTxId, 9001), "Old event should be cleaned up")
		assert.True(t, testTracker.isEventProcessed(newEvent.TransactionId, newEvent.CharacterId), "New event should be tracked")
		assert.Equal(t, 1, testTracker.getProcessedEventCount(), "Should have only new event after cleanup")
	})
}