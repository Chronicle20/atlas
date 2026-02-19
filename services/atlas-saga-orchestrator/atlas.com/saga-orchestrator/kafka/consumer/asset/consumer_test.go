package asset

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestHandleAssetCreatedEvent_WrongType(t *testing.T) {
	// Setup
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create an event with wrong type - should return early
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          "wrong_type", // Not StatusEventTypeCreated
		TransactionId: uuid.New(),
		CharacterId:   12345,
	}

	// Execute - should return early without processing
	handleAssetCreatedEvent(logger, tctx, event)

	// No assertions needed - function should return early
}

func TestHandleAssetCreatedEvent_SagaNotFound(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create an event with a transaction ID that doesn't exist
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: uuid.New(), // This saga doesn't exist
		CharacterId:   12345,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify debug log for saga not found
	var foundDebugLog bool
	for _, entry := range hook.Entries {
		if entry.Level == logrus.DebugLevel && entry.Message == "Unable to locate saga for asset created event." {
			foundDebugLog = true
			break
		}
	}
	assert.True(t, foundDebugLog, "Should have debug log for saga not found")
}

func TestHandleAssetCreatedEvent_RegularCreation(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a regular AwardInventory step (not CreateAndEquipAsset)
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("asset-creation-test").
		AddStep("award-step", saga.Pending, saga.AwardInventory, saga.AwardItemActionPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1000000,
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(te.Id(), testSaga)

	// Create an asset created event
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   12345,
		TemplateId:    1000000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify no unexpected error logs (Kafka topic errors are expected in test env)
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel {
			// Skip expected Kafka-related errors in test environment
			if entry.Message == "Unable to emit event on topic [EVENT_TOPIC_SAGA_STATUS]." ||
				entry.Message == "Failed to emit saga completion event." {
				continue
			}
			t.Errorf("Unexpected error log: %s", entry.Message)
		}
	}
}

func TestHandleAssetCreatedEvent_CreateAndEquipAsset(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a CreateAndEquipAsset step
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("create-and-equip-test").
		AddStep("create-and-equip-step", saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1302000, // Equipment item
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(te.Id(), testSaga)

	// Create an asset created event
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   12345,
		TemplateId:    1302000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify the auto-equip step was added
	var foundAutoEquipLog bool
	for _, entry := range hook.Entries {
		if entry.Message == "Successfully added auto-equip step for CreateAndEquipAsset action to be executed next." {
			foundAutoEquipLog = true
			assert.Contains(t, entry.Data, "auto_equip_step_id")
			assert.Contains(t, entry.Data, "inventory_type")
			assert.Equal(t, uint32(12345), entry.Data["character_id"])
			break
		}
	}
	assert.True(t, foundAutoEquipLog, "Should have log for auto-equip step addition")
}

func TestHandleAssetCreatedEvent_CharacterMismatch(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a CreateAndEquipAsset step for character 12345
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("mismatch-test").
		AddStep("create-and-equip-step", saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1302000,
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(te.Id(), testSaga)

	// Create an event with a different character ID
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   99999, // Different character
		TemplateId:    1302000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify error log for character mismatch
	var foundMismatchLog bool
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel && entry.Message == "Character ID mismatch in CreateAndEquipAsset creation event." {
			foundMismatchLog = true
			break
		}
	}
	assert.True(t, foundMismatchLog, "Should have error log for character mismatch")
}

func TestHandleAssetQuantityUpdatedEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expectProcess bool
	}{
		{
			name:          "Process quantity changed event",
			eventType:     asset2.StatusEventTypeQuantityChanged,
			expectProcess: true,
		},
		{
			name:          "Skip wrong event type",
			eventType:     "wrong_type",
			expectProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			event := asset2.StatusEvent[asset2.QuantityChangedEventBody]{
				Type:          tt.eventType,
				TransactionId: uuid.New(),
				CharacterId:   12345,
			}

			// Execute - function should complete without panic
			handleAssetQuantityUpdatedEvent(logger, tctx, event)
		})
	}
}

func TestHandleAssetMovedEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expectProcess bool
	}{
		{
			name:          "Process moved event",
			eventType:     asset2.StatusEventTypeMoved,
			expectProcess: true,
		},
		{
			name:          "Skip wrong event type",
			eventType:     "wrong_type",
			expectProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			event := asset2.StatusEvent[asset2.MovedStatusEventBody]{
				Type:          tt.eventType,
				TransactionId: uuid.New(),
				CharacterId:   12345,
			}

			// Execute - function should complete without panic
			handleAssetMovedEvent(logger, tctx, event)
		})
	}
}
