package character

import (
	"context"
	"testing"
	"time"

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