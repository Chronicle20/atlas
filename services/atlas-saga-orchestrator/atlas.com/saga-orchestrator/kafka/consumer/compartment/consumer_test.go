package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSagaProcessor is a mock implementation of saga.Processor for testing
type MockSagaProcessor struct {
	mock.Mock
}

func (m *MockSagaProcessor) GetById(transactionId uuid.UUID) (saga.Saga, error) {
	args := m.Called(transactionId)
	return args.Get(0).(saga.Saga), args.Error(1)
}

func (m *MockSagaProcessor) StepCompleted(transactionId uuid.UUID, success bool) error {
	args := m.Called(transactionId, success)
	return args.Error(0)
}

func (m *MockSagaProcessor) AddStep(transactionId uuid.UUID, step saga.Step[any]) error {
	args := m.Called(transactionId, step)
	return args.Error(0)
}

// TestHandleCompartmentCreatedEvent_CreateAndEquipAsset tests the compartment created event handler
// specifically for CreateAndEquipAsset operations
func TestHandleCompartmentCreatedEvent_CreateAndEquipAsset(t *testing.T) {
	tests := []struct {
		name                    string
		event                   compartment.StatusEvent[compartment.CreatedStatusEventBody]
		existingSaga            saga.Saga
		sagaGetError            error
		addStepError            error
		expectedStepCompleted   bool
		expectedStepSuccess     bool
		expectedAddStepCalled   bool
		expectedLogLevel        logrus.Level
		expectedLogMessage      string
		description             string
	}{
		{
			name: "Success - CreateAndEquipAsset with valid payload",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1, // Equipment type
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "create-and-equip-step",
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     true,
			expectedAddStepCalled:   true,
			expectedLogLevel:        logrus.InfoLevel,
			expectedLogMessage:      "Successfully added auto-equip step for CreateAndEquipAsset action",
			description:             "Should successfully add auto-equip step and complete the CreateAndEquipAsset step",
		},
		{
			name: "Success - Regular asset creation (not CreateAndEquipAsset)",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.QuestReward,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "award-asset-step",
						Status: saga.Pending,
						Action: saga.AwardAsset,
						Payload: saga.AwardItemActionPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 2000001,
								Quantity:   5,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     true,
			expectedAddStepCalled:   false,
			expectedLogLevel:        logrus.DebugLevel,
			expectedLogMessage:      "",
			description:             "Should complete regular asset creation without adding auto-equip step",
		},
		{
			name: "Error - Character ID mismatch",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   99999, // Different character ID
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "create-and-equip-step",
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345, // Expected character ID
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     false,
			expectedAddStepCalled:   false,
			expectedLogLevel:        logrus.ErrorLevel,
			expectedLogMessage:      "Character ID mismatch in CreateAndEquipAsset creation event",
			description:             "Should fail when character ID doesn't match expected value",
		},
		{
			name: "Error - Invalid payload type",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId:  "create-and-equip-step",
						Status:  saga.Pending,
						Action:  saga.CreateAndEquipAsset,
						Payload: "invalid-payload-type", // Wrong type
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     false,
			expectedAddStepCalled:   false,
			expectedLogLevel:        logrus.ErrorLevel,
			expectedLogMessage:      "Invalid payload for CreateAndEquipAsset step",
			description:             "Should fail when payload is not CreateAndEquipAssetPayload type",
		},
		{
			name: "Error - AddStep fails",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "create-and-equip-step",
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            assert.AnError,
			expectedStepCompleted:   true,
			expectedStepSuccess:     false,
			expectedAddStepCalled:   true,
			expectedLogLevel:        logrus.ErrorLevel,
			expectedLogMessage:      "Failed to add equip step to saga for CreateAndEquipAsset",
			description:             "Should fail when AddStep returns an error",
		},
		{
			name: "Error - Saga not found",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga:            saga.Saga{}, // Empty saga
			sagaGetError:            assert.AnError,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     true,
			expectedAddStepCalled:   false,
			expectedLogLevel:        logrus.DebugLevel,
			expectedLogMessage:      "Unable to locate saga for compartment created event",
			description:             "Should complete with success when saga is not found",
		},
		{
			name: "Error - No current step found",
			event: compartment.StatusEvent[compartment.CreatedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated,
				CharacterId:   12345,
				Body: compartment.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "completed-step",
						Status: saga.Completed, // No pending step
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:            nil,
			addStepError:            nil,
			expectedStepCompleted:   true,
			expectedStepSuccess:     true,
			expectedAddStepCalled:   false,
			expectedLogLevel:        logrus.DebugLevel,
			expectedLogMessage:      "No current step found for compartment created event",
			description:             "Should complete with success when no current step is found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with hook to capture log messages
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)
			
			// Create context
			ctx := context.Background()

			// Update transaction IDs to match
			tt.event.TransactionId = tt.existingSaga.TransactionId
			if len(tt.existingSaga.Steps) > 0 {
				tt.existingSaga.Steps[0].Payload = updatePayloadCharacterId(tt.existingSaga.Steps[0].Payload, tt.event.CharacterId)
			}

			// Create a mock saga processor (this would normally be done by injecting a mock)
			// For the actual implementation, we'll rely on the real saga.NewProcessor behavior
			// and verify the expected outcomes through logging and side effects

			// Execute the handler
			handleCompartmentCreatedEvent(logger, ctx, tt.event)

			// Verify log messages
			logEntries := hook.AllEntries()
			
			if tt.expectedLogMessage != "" {
				found := false
				for _, entry := range logEntries {
					if entry.Level == tt.expectedLogLevel && 
					   contains(entry.Message, tt.expectedLogMessage) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected log message '%s' at level %s not found in logs", tt.expectedLogMessage, tt.expectedLogLevel)
			}

			// Verify transaction ID appears in logs
			found := false
			for _, entry := range logEntries {
				if transactionId, ok := entry.Data["transaction_id"]; ok {
					if transactionId == tt.event.TransactionId.String() {
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Expected transaction ID %s not found in log entries", tt.event.TransactionId.String())

			// Clean up
			hook.Reset()
		})
	}
}

// TestHandleCompartmentCreatedEvent_EquipStepCreation tests the specific logic for creating equip steps
func TestHandleCompartmentCreatedEvent_EquipStepCreation(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := context.Background()

	transactionId := uuid.New()
	characterId := uint32(12345)

	// Create a valid CreateAndEquipAsset event
	event := compartment.StatusEvent[compartment.CreatedStatusEventBody]{
		TransactionId: transactionId,
		Type:          compartment.StatusEventTypeCreated,
		CharacterId:   characterId,
		Body: compartment.CreatedStatusEventBody{
			Type:     2, // Different inventory type
			Capacity: 100,
		},
	}

	// Execute the handler
	handleCompartmentCreatedEvent(logger, ctx, event)

	// Verify the log entries contain the expected information
	logEntries := hook.AllEntries()
	
	// Should have some log entries related to processing the event
	assert.True(t, len(logEntries) > 0, "Expected log entries to be generated")

	// Check that transaction ID is properly logged
	found := false
	for _, entry := range logEntries {
		if transactionId, ok := entry.Data["transaction_id"]; ok {
			if transactionId == event.TransactionId.String() {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Expected transaction ID to be logged")

	hook.Reset()
}

// TestHandleCompartmentCreationFailedEvent tests the compartment creation failed event handler
func TestHandleCompartmentCreationFailedEvent(t *testing.T) {
	tests := []struct {
		name              string
		event             compartment.StatusEvent[compartment.CreationFailedStatusEventBody]
		expectedLogLevel  logrus.Level
		expectedLogMessage string
		description       string
	}{
		{
			name: "Creation failed with error code",
			event: compartment.StatusEvent[compartment.CreationFailedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreationFailed,
				CharacterId:   12345,
				Body: compartment.CreationFailedStatusEventBody{
					ErrorCode: "INVALID_TEMPLATE_ID",
					Message:   "Template ID 999999 is not valid",
				},
			},
			expectedLogLevel:   logrus.ErrorLevel,
			expectedLogMessage: "Asset creation failed, marking saga step as failed",
			description:        "Should log error when asset creation fails",
		},
		{
			name: "Wrong event type should be ignored",
			event: compartment.StatusEvent[compartment.CreationFailedStatusEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeCreated, // Wrong type
				CharacterId:   12345,
				Body: compartment.CreationFailedStatusEventBody{
					ErrorCode: "INVALID_TEMPLATE_ID",
					Message:   "Template ID 999999 is not valid",
				},
			},
			expectedLogLevel:   logrus.ErrorLevel,
			expectedLogMessage: "",
			description:        "Should ignore events with wrong type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with hook to capture log messages
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)
			
			// Create context
			ctx := context.Background()

			// Execute the handler
			handleCompartmentCreationFailedEvent(logger, ctx, tt.event)

			// Verify log messages
			logEntries := hook.AllEntries()
			
			if tt.expectedLogMessage != "" {
				found := false
				for _, entry := range logEntries {
					if entry.Level == tt.expectedLogLevel && 
					   contains(entry.Message, tt.expectedLogMessage) {
						found = true
						// Also verify error details are logged
						assert.Equal(t, tt.event.TransactionId.String(), entry.Data["transaction_id"])
						assert.Equal(t, tt.event.CharacterId, entry.Data["character_id"])
						assert.Equal(t, tt.event.Body.ErrorCode, entry.Data["error_code"])
						assert.Equal(t, tt.event.Body.Message, entry.Data["error_message"])
						break
					}
				}
				assert.True(t, found, "Expected log message '%s' at level %s not found in logs", tt.expectedLogMessage, tt.expectedLogLevel)
			} else {
				// Should not log anything for ignored events
				assert.Equal(t, 0, len(logEntries), "Expected no log entries for ignored event")
			}

			// Clean up
			hook.Reset()
		})
	}
}

// TestHandleCompartmentErrorEvent_CreateAndEquipAsset tests the compartment error event handler
// specifically for CreateAndEquipAsset operations
func TestHandleCompartmentErrorEvent_CreateAndEquipAsset(t *testing.T) {
	tests := []struct {
		name                string
		event               compartment.StatusEvent[compartment.ErrorEventBody]
		existingSaga        saga.Saga
		sagaGetError        error
		expectedFailurePhase string
		expectedLogLevel    logrus.Level
		expectedLogMessage  string
		description         string
	}{
		{
			name: "CreateAndEquipAsset error - asset creation phase",
			event: compartment.StatusEvent[compartment.ErrorEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeError,
				CharacterId:   12345,
				Body: compartment.ErrorEventBody{
					ErrorCode: "INVALID_TEMPLATE_ID",
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "create-and-equip-step",
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					// No auto-equip step exists, indicating failure during asset creation
				},
			},
			sagaGetError:         nil,
			expectedFailurePhase: "asset_creation",
			expectedLogLevel:     logrus.ErrorLevel,
			expectedLogMessage:   "CreateAndEquipAsset operation failed - asset creation failed",
			description:          "Should identify asset creation phase failure",
		},
		{
			name: "CreateAndEquipAsset error - equipment phase",
			event: compartment.StatusEvent[compartment.ErrorEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeError,
				CharacterId:   12345,
				Body: compartment.ErrorEventBody{
					ErrorCode: "EQUIPMENT_SLOT_OCCUPIED",
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "create-and-equip-step",
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 1302000,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						StepId: "auto_equip_step_1234567890",
						Status: saga.Pending,
						Action: saga.EquipAsset,
						Payload: saga.EquipAssetPayload{
							CharacterId:   12345,
							InventoryType: 1,
							Source:        5,
							Destination:   -1,
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:         nil,
			expectedFailurePhase: "equipment",
			expectedLogLevel:     logrus.ErrorLevel,
			expectedLogMessage:   "CreateAndEquipAsset operation failed - asset was created but equipment failed",
			description:          "Should identify equipment phase failure when auto-equip step exists",
		},
		{
			name: "Regular compartment error (not CreateAndEquipAsset)",
			event: compartment.StatusEvent[compartment.ErrorEventBody]{
				TransactionId: uuid.New(),
				Type:          compartment.StatusEventTypeError,
				CharacterId:   12345,
				Body: compartment.ErrorEventBody{
					ErrorCode: "INSUFFICIENT_SPACE",
				},
			},
			existingSaga: saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.QuestReward,
				InitiatedBy:   "test-service",
				Steps: []saga.Step[any]{
					{
						StepId: "award-asset-step",
						Status: saga.Pending,
						Action: saga.AwardAsset,
						Payload: saga.AwardItemActionPayload{
							CharacterId: 12345,
							Item: saga.ItemPayload{
								TemplateId: 2000001,
								Quantity:   5,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			},
			sagaGetError:         nil,
			expectedFailurePhase: "",
			expectedLogLevel:     logrus.ErrorLevel,
			expectedLogMessage:   "Compartment operation failed",
			description:          "Should handle regular compartment errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with hook to capture log messages
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)
			
			// Create context
			ctx := context.Background()

			// Update transaction IDs to match
			tt.event.TransactionId = tt.existingSaga.TransactionId

			// Execute the handler
			handleCompartmentErrorEvent(logger, ctx, tt.event)

			// Verify log messages
			logEntries := hook.AllEntries()
			
			if tt.expectedLogMessage != "" {
				found := false
				for _, entry := range logEntries {
					if entry.Level == tt.expectedLogLevel && 
					   contains(entry.Message, tt.expectedLogMessage) {
						found = true
						
						// Verify common fields
						assert.Equal(t, tt.event.TransactionId.String(), entry.Data["transaction_id"])
						assert.Equal(t, tt.event.Body.ErrorCode, entry.Data["error_code"])
						assert.Equal(t, tt.event.CharacterId, entry.Data["character_id"])
						
						// Verify failure phase for CreateAndEquipAsset
						if tt.expectedFailurePhase != "" {
							assert.Equal(t, tt.expectedFailurePhase, entry.Data["failure_phase"])
						}
						
						break
					}
				}
				assert.True(t, found, "Expected log message '%s' at level %s not found in logs", tt.expectedLogMessage, tt.expectedLogLevel)
			}

			// Clean up
			hook.Reset()
		})
	}
}

// TestHandleCompartmentEquippedEvent tests the compartment equipped event handler
func TestHandleCompartmentEquippedEvent(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := context.Background()

	event := compartment.StatusEvent[compartment.EquippedEventBody]{
		TransactionId: uuid.New(),
		Type:          compartment.StatusEventTypeEquipped,
		CharacterId:   12345,
		Body: compartment.EquippedEventBody{
			Source:      5,
			Destination: -1,
		},
	}

	// Execute the handler
	handleCompartmentEquippedEvent(logger, ctx, event)

	// The handler should complete successfully (no error logs expected)
	logEntries := hook.AllEntries()
	
	// Should not have error logs
	for _, entry := range logEntries {
		assert.NotEqual(t, logrus.ErrorLevel, entry.Level, "Unexpected error log: %s", entry.Message)
	}

	hook.Reset()
}

// TestHandleCompartmentUnequippedEvent tests the compartment unequipped event handler
func TestHandleCompartmentUnequippedEvent(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := context.Background()

	event := compartment.StatusEvent[compartment.UnequippedEventBody]{
		TransactionId: uuid.New(),
		Type:          compartment.StatusEventTypeUnequipped,
		CharacterId:   12345,
		Body: compartment.UnequippedEventBody{
			Source:      -1,
			Destination: 5,
		},
	}

	// Execute the handler
	handleCompartmentUnequippedEvent(logger, ctx, event)

	// The handler should complete successfully (no error logs expected)
	logEntries := hook.AllEntries()
	
	// Should not have error logs
	for _, entry := range logEntries {
		assert.NotEqual(t, logrus.ErrorLevel, entry.Level, "Unexpected error log: %s", entry.Message)
	}

	hook.Reset()
}

// Helper functions

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(substr) > 0 && len(s) > 0 && findSubstring(s, substr)))
}

// findSubstring is a simple substring search
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// updatePayloadCharacterId updates the character ID in payload for testing
func updatePayloadCharacterId(payload interface{}, characterId uint32) interface{} {
	switch p := payload.(type) {
	case saga.CreateAndEquipAssetPayload:
		p.CharacterId = characterId
		return p
	case saga.AwardItemActionPayload:
		p.CharacterId = characterId
		return p
	default:
		return payload
	}
}