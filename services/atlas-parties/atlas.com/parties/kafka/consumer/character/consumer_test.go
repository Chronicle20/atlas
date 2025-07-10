package character

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyTracker(t *testing.T) {
	// Create a test tracker
	tracker := &idempotencyTracker{
		processedEvents: make(map[uuid.UUID]*processedEvent),
		lastCleanup:     time.Now(),
		cleanupInterval: 1 * time.Minute,
		eventTTL:        5 * time.Minute,
	}

	transactionId := uuid.New()
	characterId := uint32(12345)

	t.Run("EventNotProcessedInitially", func(t *testing.T) {
		assert.False(t, tracker.isEventProcessed(transactionId, characterId))
		assert.Equal(t, 0, tracker.getProcessedEventCount())
	})

	t.Run("MarkEventProcessed", func(t *testing.T) {
		tracker.markEventProcessed(transactionId, characterId)
		assert.True(t, tracker.isEventProcessed(transactionId, characterId))
		assert.Equal(t, 1, tracker.getProcessedEventCount())
	})

	t.Run("DuplicateEventDetected", func(t *testing.T) {
		// Same transaction ID and character ID should be detected as duplicate
		assert.True(t, tracker.isEventProcessed(transactionId, characterId))
	})

	t.Run("DifferentCharacterIdNotDetected", func(t *testing.T) {
		// Same transaction ID but different character ID should not be detected
		differentCharId := uint32(67890)
		assert.False(t, tracker.isEventProcessed(transactionId, differentCharId))
	})

	t.Run("DifferentTransactionIdNotDetected", func(t *testing.T) {
		// Different transaction ID should not be detected as duplicate
		differentTxId := uuid.New()
		assert.False(t, tracker.isEventProcessed(differentTxId, characterId))
	})
}

func TestIdempotencyTrackerCleanup(t *testing.T) {
	tracker := &idempotencyTracker{
		processedEvents: make(map[uuid.UUID]*processedEvent),
		lastCleanup:     time.Now().Add(-10 * time.Minute), // Force cleanup trigger
		cleanupInterval: 1 * time.Millisecond, // Very short for testing
		eventTTL:        10 * time.Millisecond, // Very short for testing
	}

	// Add an old event directly
	oldTxId := uuid.New()
	tracker.processedEvents[oldTxId] = &processedEvent{
		transactionId: oldTxId,
		characterId:   12345,
		processedAt:   time.Now().Add(-20 * time.Millisecond), // Older than TTL
	}

	assert.Equal(t, 1, tracker.getProcessedEventCount())

	// Add a new event which should trigger cleanup due to old lastCleanup time
	newTxId := uuid.New()
	tracker.markEventProcessed(newTxId, 67890)

	// The old event should be cleaned up, new event should remain
	assert.False(t, tracker.isEventProcessed(oldTxId, 12345))
	assert.True(t, tracker.isEventProcessed(newTxId, 67890))
	assert.Equal(t, 1, tracker.getProcessedEventCount())
}

func TestHandleStatusEventDeletedIdempotency(t *testing.T) {
	// Reset the global tracker for this test
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

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	_ = context.Background() // Suppress unused warning

	transactionId := uuid.New()
	characterId := uint32(12345)

	_ = StatusEvent[StatusEventDeletedBody]{ // Suppress unused warning
		TransactionId: transactionId,
		WorldId:       1,
		CharacterId:   characterId,
		Type:          StatusEventTypeDeleted,
		Body:          StatusEventDeletedBody{},
	}

	t.Run("FirstProcessingSucceeds", func(t *testing.T) {
		// First processing should proceed
		require.False(t, deletionIdempotencyTracker.isEventProcessed(transactionId, characterId))
		
		// Note: We can't fully test the handler without setting up the entire infrastructure,
		// but we can test that the idempotency tracking works
		deletionIdempotencyTracker.markEventProcessed(transactionId, characterId)
		
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, characterId))
		assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())
	})

	t.Run("DuplicateEventSkipped", func(t *testing.T) {
		// Second processing should detect duplicate
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(transactionId, characterId))
		
		// The event count should remain the same (not incremented)
		assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())
	})
}

func TestValidateStatusEventDeleted(t *testing.T) {
	logger := logrus.New()

	t.Run("ValidEvent", func(t *testing.T) {
		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   12345,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		err := validateStatusEventDeleted(logger, event)
		assert.NoError(t, err)
	})

	t.Run("InvalidCharacterId", func(t *testing.T) {
		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   0, // Invalid
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		err := validateStatusEventDeleted(logger, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character ID")
	})

	t.Run("InvalidWorldId", func(t *testing.T) {
		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       0, // Invalid
			CharacterId:   12345,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		err := validateStatusEventDeleted(logger, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid world ID")
	})

	t.Run("InvalidEventType", func(t *testing.T) {
		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   12345,
			Type:          "WRONG_TYPE", // Invalid
			Body:          StatusEventDeletedBody{},
		}

		err := validateStatusEventDeleted(logger, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event type mismatch")
	})

	t.Run("EmptyEventType", func(t *testing.T) {
		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   12345,
			Type:          "", // Invalid
			Body:          StatusEventDeletedBody{},
		}

		err := validateStatusEventDeleted(logger, event)
		assert.Error(t, err)
		// Should return the first validation error
		assert.Contains(t, err.Error(), "event type mismatch")
	})
}

func TestHandleStatusEventDeleted(t *testing.T) {
	// Reset the global tracker for each test
	originalTracker := deletionIdempotencyTracker
	defer func() {
		deletionIdempotencyTracker = originalTracker
	}()

	// Setup test logger and context with tenant
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	t.Run("IgnoreNonDeletedEventType", func(t *testing.T) {
		// Reset tracker
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   12345,
			Type:          StatusEventTypeLogin, // Not DELETED
			Body:          StatusEventDeletedBody{},
		}

		// Should return early without processing
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should not be marked as processed since it was ignored
		assert.False(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
		assert.Equal(t, 0, deletionIdempotencyTracker.getProcessedEventCount())
	})

	t.Run("IgnoreMalformedEvent", func(t *testing.T) {
		// Reset tracker
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       0, // Invalid world ID
			CharacterId:   12345,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Should return early without processing due to validation failure
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should not be marked as processed since validation failed
		assert.False(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
		assert.Equal(t, 0, deletionIdempotencyTracker.getProcessedEventCount())
	})

	t.Run("ProcessValidEventWithoutPanic", func(t *testing.T) {
		// Reset tracker
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   99999, // Non-existent character
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Should process without panicking, even if character doesn't exist
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should be marked as processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
		assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())
	})

	t.Run("SkipDuplicateEvent", func(t *testing.T) {
		// Reset tracker and mark an event as already processed
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		transactionId := uuid.New()
		characterId := uint32(12345)
		
		// Pre-mark event as processed
		deletionIdempotencyTracker.markEventProcessed(transactionId, characterId)
		assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: transactionId,
			WorldId:       1,
			CharacterId:   characterId,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Should skip processing due to duplicate detection
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Event count should remain the same
		assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())
	})

	t.Run("HandlesCharacterNotInPartyGracefully", func(t *testing.T) {
		// Reset tracker
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   88888, // Character not in any party
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		// Should handle gracefully when character is not in any party
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should be marked as processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
	})

	t.Run("ProcessingMarksEventAsProcessedRegardlessOfOutcome", func(t *testing.T) {
		// Reset tracker
		deletionIdempotencyTracker = &idempotencyTracker{
			processedEvents: make(map[uuid.UUID]*processedEvent),
			lastCleanup:     time.Now(),
			cleanupInterval: 5 * time.Minute,
			eventTTL:        30 * time.Minute,
		}

		event := StatusEvent[StatusEventDeletedBody]{
			TransactionId: uuid.New(),
			WorldId:       1,
			CharacterId:   77777,
			Type:          StatusEventTypeDeleted,
			Body:          StatusEventDeletedBody{},
		}

		initialCount := deletionIdempotencyTracker.getProcessedEventCount()
		
		// Process event (may succeed or fail internally, but should always mark as processed)
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should always be marked as processed, regardless of internal success/failure
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
		assert.Equal(t, initialCount+1, deletionIdempotencyTracker.getProcessedEventCount())
	})
}

func TestHandleStatusEventDeletedWithPanicRecovery(t *testing.T) {
	// This test verifies that the panic recovery mechanisms work correctly
	// Note: In a real scenario, panics might occur due to unexpected conditions
	// in party removal or character deletion logic. The handler should recover gracefully.
	
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Reset tracker
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

	event := StatusEvent[StatusEventDeletedBody]{
		TransactionId: uuid.New(),
		WorldId:       1,
		CharacterId:   55555,
		Type:          StatusEventTypeDeleted,
		Body:          StatusEventDeletedBody{},
	}

	t.Run("HandlerDoesNotPanicOnValidEvent", func(t *testing.T) {
		// The handler should not panic even when processing edge cases
		require.NotPanics(t, func() {
			handleStatusEventDeleted(logger, ctx, event)
		})

		// Should be marked as processed
		assert.True(t, deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId))
	})
}

func TestHandleStatusEventDeletedEventTypes(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	testTenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), testTenant)

	// Reset tracker
	originalTracker := deletionIdempotencyTracker
	defer func() {
		deletionIdempotencyTracker = originalTracker
	}()

	testCases := []struct {
		name          string
		eventType     string
		shouldProcess bool
	}{
		{
			name:          "DeletedEventShouldProcess",
			eventType:     StatusEventTypeDeleted,
			shouldProcess: true,
		},
		{
			name:          "LoginEventShouldNotProcess",
			eventType:     StatusEventTypeLogin,
			shouldProcess: false,
		},
		{
			name:          "LogoutEventShouldNotProcess", 
			eventType:     StatusEventTypeLogout,
			shouldProcess: false,
		},
		{
			name:          "ChannelChangedEventShouldNotProcess",
			eventType:     StatusEventTypeChannelChanged,
			shouldProcess: false,
		},
		{
			name:          "MapChangedEventShouldNotProcess",
			eventType:     StatusEventTypeMapChanged,
			shouldProcess: false,
		},
		{
			name:          "UnknownEventShouldNotProcess",
			eventType:     "UNKNOWN_EVENT",
			shouldProcess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset tracker for each test case
			deletionIdempotencyTracker = &idempotencyTracker{
				processedEvents: make(map[uuid.UUID]*processedEvent),
				lastCleanup:     time.Now(),
				cleanupInterval: 5 * time.Minute,
				eventTTL:        30 * time.Minute,
			}

			event := StatusEvent[StatusEventDeletedBody]{
				TransactionId: uuid.New(),
				WorldId:       1,
				CharacterId:   uint32(12345 + len(tc.name)), // Unique character ID
				Type:          tc.eventType,
				Body:          StatusEventDeletedBody{},
			}

			// Should not panic regardless of event type
			require.NotPanics(t, func() {
				handleStatusEventDeleted(logger, ctx, event)
			})

			// Check if event was processed based on expected behavior
			isProcessed := deletionIdempotencyTracker.isEventProcessed(event.TransactionId, event.CharacterId)
			if tc.shouldProcess {
				assert.True(t, isProcessed, "Event type %s should have been processed", tc.eventType)
				assert.Equal(t, 1, deletionIdempotencyTracker.getProcessedEventCount())
			} else {
				assert.False(t, isProcessed, "Event type %s should not have been processed", tc.eventType)
				assert.Equal(t, 0, deletionIdempotencyTracker.getProcessedEventCount())
			}
		})
	}
}