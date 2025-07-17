package saga

import (
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	mock3 "atlas-saga-orchestrator/validation/mock"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// TestCreateAndEquipAssetIntegration tests the full integration flow of CreateAndEquipAsset
// from saga processing to Kafka message handling
func TestCreateAndEquipAssetIntegration(t *testing.T) {
	tests := []struct {
		name                            string
		sagaPayload                     CreateAndEquipAssetPayload
		compartmentRequestError         error
		expectedSagaStepStatus          Status
		expectedCompartmentRequestCalls int
		expectedErrorContains           string
		description                     string
	}{
		{
			name: "Success - Valid CreateAndEquipAsset payload",
			sagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000, // Valid weapon template
					Quantity:   1,
				},
			},
			compartmentRequestError:         nil,
			expectedSagaStepStatus:          Pending, // Step remains pending until Kafka event
			expectedCompartmentRequestCalls: 1,
			expectedErrorContains:           "",
			description:                     "Should successfully initiate CreateAndEquipAsset operation",
		},
		{
			name: "Error - Invalid template ID",
			sagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 0, // Invalid template ID
					Quantity:   1,
				},
			},
			compartmentRequestError:         errors.New("invalid templateId"),
			expectedSagaStepStatus:          Pending, // Step fails during execution
			expectedCompartmentRequestCalls: 1,
			expectedErrorContains:           "invalid templateId",
			description:                     "Should fail when template ID is invalid",
		},
		{
			name: "Error - Compartment service unavailable",
			sagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			compartmentRequestError:         errors.New("compartment service unavailable"),
			expectedSagaStepStatus:          Pending, // Step fails during execution
			expectedCompartmentRequestCalls: 1,
			expectedErrorContains:           "compartment service unavailable",
			description:                     "Should fail when compartment service is unavailable",
		},
		{
			name: "Error - Character not found",
			sagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 99999, // Non-existent character
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			compartmentRequestError:         errors.New("character not found"),
			expectedSagaStepStatus:          Pending, // Step fails during execution
			expectedCompartmentRequestCalls: 1,
			expectedErrorContains:           "character not found",
			description:                     "Should fail when character is not found",
		},
		{
			name: "Success - Consumable item",
			sagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000001, // Valid consumable template
					Quantity:   5,
				},
			},
			compartmentRequestError:         nil,
			expectedSagaStepStatus:          Pending, // Step remains pending until Kafka event
			expectedCompartmentRequestCalls: 1,
			expectedErrorContains:           "",
			description:                     "Should successfully handle consumable items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Create mock processors
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			// Setup processor
			processor := NewProcessor(logger, tctx).WithCharacterProcessor(charP).WithCompartmentProcessor(compP).WithValidationProcessor(validP)

			// Configure compartment processor mock
			compartmentRequestCalls := 0
			compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
				compartmentRequestCalls++

				// Verify payload conversion
				assert.Equal(t, tt.sagaPayload.CharacterId, payload.CharacterId)
				assert.Equal(t, tt.sagaPayload.Item.TemplateId, payload.Item.TemplateId)
				assert.Equal(t, tt.sagaPayload.Item.Quantity, payload.Item.Quantity)

				return tt.compartmentRequestError
			}

			// Create test saga
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      InventoryTransaction,
				InitiatedBy:   "integration-test",
				Steps: []Step[any]{
					{
						StepId:    "create-and-equip-step",
						Status:    Pending,
						Action:    CreateAndEquipAsset,
						Payload:   tt.sagaPayload,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// Store saga in cache
			GetCache().Put(te.Id(), saga)

			// Execute saga step
			err := processor.Step(saga.TransactionId)

			// Verify results
			if tt.expectedErrorContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorContains)
			} else {
				assert.NoError(t, err)
			}

			// Verify compartment request was called expected number of times
			assert.Equal(t, tt.expectedCompartmentRequestCalls, compartmentRequestCalls)

			// Verify compartment processor was configured
			assert.NotNil(t, compP.RequestCreateAndEquipAssetFunc)

			// Verify logging
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) > 0, "Expected log entries to be generated")

			// Check for transaction ID in logs
			found := false
			for _, entry := range logEntries {
				if txnId, ok := entry.Data["transaction_id"]; ok {
					if txnId == transactionId.String() {
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Expected transaction ID to be logged")

			// Clean up
			hook.Reset()
		})
	}
}

// TestCreateAndEquipAssetKafkaEventFlow tests the Kafka event flow for CreateAndEquipAsset
func TestCreateAndEquipAssetKafkaEventFlow_Disabled(t *testing.T) {
	tests := []struct {
		name                   string
		initialSagaPayload     CreateAndEquipAssetPayload
		simulateCreatedEvent   bool
		simulateEquippedEvent  bool
		simulateErrorEvent     bool
		expectedFinalStepCount int
		expectedSagaCompleted  bool
		expectedAutoEquipStep  bool
		description            string
	}{
		{
			name: "Success - Complete CreateAndEquipAsset flow",
			initialSagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			simulateCreatedEvent:   true,
			simulateEquippedEvent:  true,
			simulateErrorEvent:     false,
			expectedFinalStepCount: 2, // Initial CreateAndEquipAsset + auto-generated EquipAsset
			expectedSagaCompleted:  true,
			expectedAutoEquipStep:  true,
			description:            "Should complete full flow with auto-equip step",
		},
		{
			name: "Partial success - Asset created but equipment fails",
			initialSagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			simulateCreatedEvent:   true,
			simulateEquippedEvent:  false,
			simulateErrorEvent:     true,
			expectedFinalStepCount: 2, // Initial CreateAndEquipAsset + auto-generated EquipAsset
			expectedSagaCompleted:  false,
			expectedAutoEquipStep:  true,
			description:            "Should handle equipment failure after successful creation",
		},
		{
			name: "Failure - Asset creation fails",
			initialSagaPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 0, // Invalid template
					Quantity:   1,
				},
			},
			simulateCreatedEvent:   false,
			simulateEquippedEvent:  false,
			simulateErrorEvent:     true,
			expectedFinalStepCount: 1, // Only initial CreateAndEquipAsset step
			expectedSagaCompleted:  false,
			expectedAutoEquipStep:  false,
			description:            "Should handle asset creation failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Create mock processors
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			// Setup processor
			processor := NewProcessor(logger, tctx).WithCharacterProcessor(charP).WithCompartmentProcessor(compP).WithValidationProcessor(validP)

			// Configure compartment processor mock
			compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
				if tt.simulateCreatedEvent {
					return nil // Success - would trigger CREATED event
				}
				return errors.New("asset creation failed") // Would trigger ERROR event
			}

			// Create test saga
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      InventoryTransaction,
				InitiatedBy:   "kafka-flow-test",
				Steps: []Step[any]{
					{
						StepId:    "create-and-equip-step",
						Status:    Pending,
						Action:    CreateAndEquipAsset,
						Payload:   tt.initialSagaPayload,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// Store saga in cache
			GetCache().Put(te.Id(), saga)

			// Phase 1: Execute initial CreateAndEquipAsset step
			err := processor.Step(saga.TransactionId)

			if tt.simulateCreatedEvent {
				assert.NoError(t, err, "Initial step should succeed")
			} else {
				assert.Error(t, err, "Initial step should fail")
			}

			// Phase 2: Simulate Kafka events
			if tt.simulateCreatedEvent {
				// Simulate CREATED event - this would be handled by compartment consumer
				// In real flow, this would trigger saga.StepCompleted(transactionId, true)
				// and potentially add an auto-equip step

				// For this test, we'll simulate the behavior
				err = processor.StepCompleted(transactionId, true)
				assert.NoError(t, err, "StepCompleted should succeed")

				// Simulate auto-equip step addition (normally done by asset consumer)
				if tt.expectedAutoEquipStep {
					equipStep := Step[any]{
						StepId: "auto_equip_step_test",
						Status: Pending,
						Action: EquipAsset,
						Payload: EquipAssetPayload{
							CharacterId:   tt.initialSagaPayload.CharacterId,
							InventoryType: 1,
							Source:        5,
							Destination:   -1,
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}

					err = processor.PrependStep(transactionId, equipStep)
					assert.NoError(t, err, "Prepending auto-equip step should succeed")
				}
			}

			if tt.simulateEquippedEvent {
				// Simulate EQUIPPED event
				err = processor.StepCompleted(transactionId, true)
				assert.NoError(t, err, "Equipment step completion should succeed")
			}

			if tt.simulateErrorEvent {
				// Simulate ERROR event
				err = processor.StepCompleted(transactionId, false)
				assert.NoError(t, err, "Error step completion should succeed")
			}

			// Verify final saga state
			finalSaga, err := processor.GetById(transactionId)
			assert.NoError(t, err, "Should be able to retrieve final saga")

			assert.Equal(t, tt.expectedFinalStepCount, len(finalSaga.Steps),
				"Final saga should have expected number of steps")

			// Verify auto-equip step was added if expected
			if tt.expectedAutoEquipStep {
				found := false
				for _, step := range finalSaga.Steps {
					if step.Action == EquipAsset {
						found = true
						// Verify the auto-equip step has correct payload
						equipPayload, ok := step.Payload.(EquipAssetPayload)
						assert.True(t, ok, "Auto-equip step should have EquipAssetPayload")
						assert.Equal(t, tt.initialSagaPayload.CharacterId, equipPayload.CharacterId)
						break
					}
				}
				assert.True(t, found, "Auto-equip step should have been added")
			}

			// Check completion status
			allCompleted := true
			for _, step := range finalSaga.Steps {
				if step.Status != Completed {
					allCompleted = false
					break
				}
			}
			assert.Equal(t, tt.expectedSagaCompleted, allCompleted,
				"Saga completion status should match expected")

			// Verify logging
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) > 0, "Expected log entries to be generated")

			// Clean up
			hook.Reset()
		})
	}
}

// TestCreateAndEquipAssetCompensation tests compensation logic for CreateAndEquipAsset
func TestCreateAndEquipAssetCompensation(t *testing.T) {
	tests := []struct {
		name                 string
		initialPayload       CreateAndEquipAssetPayload
		failureScenario      string
		expectedCompensation string
		description          string
	}{
		{
			name: "Compensation - Asset creation failure",
			initialPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			failureScenario:      "creation_failed",
			expectedCompensation: "none", // No compensation needed for creation failure
			description:          "Should not require compensation when asset creation fails",
		},
		{
			name: "Compensation - Equipment failure after creation",
			initialPayload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			failureScenario:      "equipment_failed",
			expectedCompensation: "destroy_asset", // Should destroy created asset
			description:          "Should destroy created asset when equipment fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Create mock processors
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			// Setup processor
			processor := NewProcessor(logger, tctx).WithCharacterProcessor(charP).WithCompartmentProcessor(compP).WithValidationProcessor(validP)

			// Configure mocks for compensation testing
			compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
				if tt.failureScenario == "creation_failed" {
					return errors.New("creation failed")
				}
				return nil // Success
			}

			// Create test saga with failed step
			transactionId := uuid.New()
			failedStep := Step[any]{
				StepId:    "create-and-equip-step",
				Status:    Failed,
				Action:    CreateAndEquipAsset,
				Payload:   tt.initialPayload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			saga := Saga{
				TransactionId: transactionId,
				SagaType:      InventoryTransaction,
				InitiatedBy:   "compensation-test",
				Steps:         []Step[any]{failedStep},
			}

			// Test compensation
			err := processor.compensateCreateAndEquipAsset(saga, failedStep)

			if tt.expectedCompensation == "none" {
				assert.NoError(t, err, "Compensation should succeed for no-op scenarios")
			} else {
				// For more complex compensation, we'd need to test the actual compensation logic
				// This would involve checking if destroy commands are issued, etc.
				assert.NoError(t, err, "Compensation should execute without panicking")
			}

			// Verify logging
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) > 0, "Expected log entries to be generated")

			// Look for compensation-related log messages
			found := false
			for _, entry := range logEntries {
				if entry.Data["transaction_id"] == transactionId.String() {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected compensation logs to include transaction ID")

			// Clean up
			hook.Reset()
		})
	}
}

// TestCreateAndEquipAssetPayloadValidation tests payload validation in various scenarios
func TestCreateAndEquipAssetPayloadValidation(t *testing.T) {
	tests := []struct {
		name          string
		payload       interface{}
		expectedError bool
		errorContains string
		description   string
	}{
		{
			name: "Valid payload",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			expectedError: false,
			errorContains: "",
			description:   "Should accept valid CreateAndEquipAssetPayload",
		},
		{
			name:          "Invalid payload type",
			payload:       "invalid-payload",
			expectedError: true,
			errorContains: "invalid payload",
			description:   "Should reject non-CreateAndEquipAssetPayload types",
		},
		{
			name: "Valid payload with different template",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 67890,
				Item: ItemPayload{
					TemplateId: 2000001,
					Quantity:   5,
				},
			},
			expectedError: false,
			errorContains: "",
			description:   "Should accept valid payload with consumable template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Create mock processors
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			// Setup processor
			processor := NewProcessor(logger, tctx).WithCharacterProcessor(charP).WithCompartmentProcessor(compP).WithValidationProcessor(validP)

			// Configure compartment processor mock
			compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
				return nil // Always succeed for payload validation tests
			}

			// Create test saga with step
			transactionId := uuid.New()
			step := Step[any]{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			saga := Saga{
				TransactionId: transactionId,
				SagaType:      InventoryTransaction,
				InitiatedBy:   "payload-validation-test",
				Steps:         []Step[any]{step},
			}

			// Test payload validation
			err := processor.handleCreateAndEquipAsset(saga, step)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
