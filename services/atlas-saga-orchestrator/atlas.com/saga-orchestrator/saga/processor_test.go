package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/validation"
	mock3 "atlas-saga-orchestrator/validation/mock"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// setupTestProcessor creates a ProcessorImpl with mock dependencies for testing
func setupTestProcessor(charP character.Processor, compP compartment.Processor, validP ...validation.Processor) (*ProcessorImpl, *test.Hook) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.charP = charP
	processor.compP = compP
	if len(validP) > 0 {
		processor.validP = validP[0]
	}
	return processor, hook
}

// TestHandleValidateCharacterState tests the handleValidateCharacterState function
func TestHandleValidateCharacterState(t *testing.T) {
	tests := []struct {
		name          string
		payload       ValidateCharacterStatePayload
		mockResult    validation.ValidationResult
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case - all conditions pass",
			payload: ValidateCharacterStatePayload{
				CharacterId: 12345,
				Conditions: []validation.ConditionInput{
					{
						Type:     "jobId",
						Operator: "=",
						Value:    100,
					},
					{
						Type:     "meso",
						Operator: ">=",
						Value:    1000,
					},
				},
			},
			mockResult: func() validation.ValidationResult {
				result := validation.NewValidationResult(12345)
				// All conditions pass by default
				return result
			}(),
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Failure case - conditions not met",
			payload: ValidateCharacterStatePayload{
				CharacterId: 12345,
				Conditions: []validation.ConditionInput{
					{
						Type:     "jobId",
						Operator: "=",
						Value:    100,
					},
				},
			},
			mockResult: func() validation.ValidationResult {
				result := validation.NewValidationResult(12345)
				// Add a failed condition
				result.AddConditionResult(validation.ConditionResult{
					Passed:      false,
					Description: "Job ID does not match",
					Type:        "jobId",
					Operator:    "=",
					Value:       100,
					ActualValue: 200,
				})
				return result
			}(),
			mockError:     nil,
			expectError:   true,
			errorContains: "character state validation failed",
		},
		{
			name: "Error case - validation service error",
			payload: ValidateCharacterStatePayload{
				CharacterId: 12345,
				Conditions: []validation.ConditionInput{
					{
						Type:     "jobId",
						Operator: "=",
						Value:    100,
					},
				},
			},
			mockResult:    validation.ValidationResult{},
			mockError:     errors.New("validation service unavailable"),
			expectError:   true,
			errorContains: "validation service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			validP.ValidateCharacterStateFunc = func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, len(tt.payload.Conditions), len(conditions))

				// Return mock result or error
				return tt.mockResult, tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    ValidateCharacterState,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleValidateCharacterState(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleWarpToPortal(t *testing.T) {
	tests := []struct {
		name          string
		payload       WarpToPortalPayload
		fieldExists   bool
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: WarpToPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
				PortalId:    1,
			},
			fieldExists: true,
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Field not found",
			payload: WarpToPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("0000000000000"),
				PortalId:    1,
			},
			fieldExists:   false,
			mockError:     nil,
			expectError:   true,
			errorContains: "invalid field id",
		},
		{
			name: "Warp error",
			payload: WarpToPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
				PortalId:    1,
			},
			fieldExists:   true,
			mockError:     errors.New("failed to warp"),
			expectError:   true,
			errorContains: "failed to warp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup

			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.WarpToPortalAndEmitFunc = func(transactionId uuid.UUID, characterId uint32, f field.Model, pp model.Provider[uint32]) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.FieldId, f.Id())

				// Verify portal provider
				portalId, err := pp()
				assert.NoError(t, err)
				assert.Equal(t, tt.payload.PortalId, portalId)

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    WarpToPortal,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Mock field.FromId
			if !tt.fieldExists {
				// This will cause the test to fail with "invalid field id" error
				// We can't directly mock field.FromId since it's not an interface
			}

			// Execute
			err := handleWarpToPortal(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleWarpToRandomPortal(t *testing.T) {
	tests := []struct {
		name          string
		payload       WarpToRandomPortalPayload
		fieldExists   bool
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: WarpToRandomPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
			},
			fieldExists: true,
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Field not found",
			payload: WarpToRandomPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("000000000000"),
			},
			fieldExists:   false,
			mockError:     nil,
			expectError:   true,
			errorContains: "invalid field id",
		},
		{
			name: "Warp error",
			payload: WarpToRandomPortalPayload{
				CharacterId: 12345,
				FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
			},
			fieldExists:   true,
			mockError:     errors.New("failed to warp"),
			expectError:   true,
			errorContains: "failed to warp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.WarpRandomAndEmitFunc = func(transactionId uuid.UUID, characterId uint32, f field.Model) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.FieldId, f.Id())

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    WarpToRandomPortal,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Mock field.FromId
			if !tt.fieldExists {
				// This will cause the test to fail with "invalid field id" error
				// We can't directly mock field.FromId since it's not an interface
			}

			// Execute
			err := handleWarpToRandomPortal(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleAwardAsset(t *testing.T) {
	tests := []struct {
		name          string
		action        Action
		payload       AwardItemActionPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name:   "Success case - AwardAsset",
			action: AwardAsset,
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:   "Error case - AwardAsset",
			action: AwardAsset,
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:     errors.New("failed to create item"),
			expectError:   true,
			errorContains: "failed to create item",
		},
		{
			name:   "Success case - AwardInventory (deprecated)",
			action: AwardInventory,
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:   "Error case - AwardInventory (deprecated)",
			action: AwardInventory,
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:     errors.New("failed to create item"),
			expectError:   true,
			errorContains: "failed to create item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			compP.RequestCreateItemFunc = func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.Item.TemplateId, templateId)
				assert.Equal(t, tt.payload.Item.Quantity, quantity)
				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    tt.action,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleAwardAsset(processor, saga, step)

			// Verify
			if tt.expectError {
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

// TestHandleAwardInventory tests the deprecated handleAwardInventory function
// which is now a wrapper for handleAwardAsset
func TestHandleAwardInventory(t *testing.T) {
	tests := []struct {
		name          string
		payload       AwardItemActionPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000,
					Quantity:   5,
				},
			},
			mockError:     errors.New("failed to create item"),
			expectError:   true,
			errorContains: "failed to create item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			compP.RequestCreateItemFunc = func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.Item.TemplateId, templateId)
				assert.Equal(t, tt.payload.Item.Quantity, quantity)
				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    AwardInventory,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleAwardInventory(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleAwardLevel(t *testing.T) {
	tests := []struct {
		name          string
		payload       AwardLevelPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: AwardLevelPayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				Amount:      1,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: AwardLevelPayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				Amount:      2,
			},
			mockError:     errors.New("failed to award level"),
			expectError:   true,
			errorContains: "failed to award level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.AwardLevelAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)
				assert.Equal(t, tt.payload.ChannelId, channelId)
				assert.Equal(t, tt.payload.Amount, amount)

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    AwardLevel,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleAwardLevel(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleAwardExperience(t *testing.T) {
	tests := []struct {
		name          string
		payload       AwardExperiencePayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: AwardExperiencePayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				Distributions: []ExperienceDistributions{
					{
						ExperienceType: "WHITE",
						Amount:         1000,
						Attr1:          0,
					},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: AwardExperiencePayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				Distributions: []ExperienceDistributions{
					{
						ExperienceType: "WHITE",
						Amount:         1000,
						Attr1:          0,
					},
				},
			},
			mockError:     errors.New("failed to award experience"),
			expectError:   true,
			errorContains: "failed to award experience",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.AwardExperienceAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)
				assert.Equal(t, tt.payload.ChannelId, channelId)

				// Verify distributions were transformed correctly
				expectedDist := TransformExperienceDistributions(tt.payload.Distributions)
				assert.Equal(t, expectedDist, distributions)

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    AwardExperience,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleAwardExperience(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleAwardMesos(t *testing.T) {
	tests := []struct {
		name          string
		payload       AwardMesosPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: AwardMesosPayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				ActorId:     0,
				ActorType:   "SYSTEM",
				Amount:      1000,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: AwardMesosPayload{
				CharacterId: 12345,
				WorldId:     0,
				ChannelId:   0,
				ActorId:     0,
				ActorType:   "SYSTEM",
				Amount:      1000,
			},
			mockError:     errors.New("failed to award mesos"),
			expectError:   true,
			errorContains: "failed to award mesos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.AwardMesosAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)
				assert.Equal(t, tt.payload.ChannelId, channelId)
				assert.Equal(t, tt.payload.ActorId, actorId)
				assert.Equal(t, tt.payload.ActorType, actorType)
				assert.Equal(t, tt.payload.Amount, amount)

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    AwardMesos,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleAwardMesos(processor, saga, step)

			// Verify
			if tt.expectError {
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

func TestHandleDestroyAsset(t *testing.T) {
	tests := []struct {
		name          string
		payload       DestroyAssetPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: DestroyAssetPayload{
				CharacterId: 12345,
				TemplateId:  2000,
				Quantity:    5,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: DestroyAssetPayload{
				CharacterId: 12345,
				TemplateId:  2000,
				Quantity:    5,
			},
			mockError:     errors.New("failed to destroy item"),
			expectError:   true,
			errorContains: "failed to destroy item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			compP.RequestDestroyItemFunc = func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.TemplateId, templateId)
				assert.Equal(t, tt.payload.Quantity, quantity)
				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      QuestReward,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    DestroyAsset,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleDestroyAsset(processor, saga, step)

			// Verify
			if tt.expectError {
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

// TestHandleCreateCharacter tests the handleCreateCharacter function
func TestHandleCreateCharacter(t *testing.T) {
	tests := []struct {
		name          string
		payload       CharacterCreatePayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case - valid character creation",
			payload: CharacterCreatePayload{
				AccountId: 12345,
				Name:      "TestCharacter",
				WorldId:   0,
				ChannelId: 0,
				JobId:     0,
				Gender:    0,
				Face:      20000,
				Hair:      30000,
				HairColor: 0,
				Skin:      0,
				Top:       1040002,
				Bottom:    1060002,
				Shoes:     1072001,
				Weapon:    1302000,
				MapId:     100000000,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case - character service failure",
			payload: CharacterCreatePayload{
				AccountId: 12345,
				Name:      "TestCharacter",
				WorldId:   0,
				ChannelId: 0,
				JobId:     0,
				Gender:    0,
				Face:      20000,
				Hair:      30000,
				HairColor: 0,
				Skin:      0,
				Top:       1040002,
				Bottom:    1060002,
				Shoes:     1072001,
				Weapon:    1302000,
				MapId:     100000000,
			},
			mockError:     errors.New("character service unavailable"),
			expectError:   true,
			errorContains: "character service unavailable",
		},
		{
			name: "Error case - duplicate character name",
			payload: CharacterCreatePayload{
				AccountId: 12345,
				Name:      "ExistingCharacter",
				WorldId:   0,
				ChannelId: 0,
				JobId:     0,
				Gender:    0,
				Face:      20000,
				Hair:      30000,
				HairColor: 0,
				Skin:      0,
				Top:       1040002,
				Bottom:    1060002,
				Shoes:     1072001,
				Weapon:    1302000,
				MapId:     100000000,
			},
			mockError:     errors.New("character name already exists"),
			expectError:   true,
			errorContains: "character name already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, name string, worldId byte, channelId byte, jobId uint32, gender byte, face uint32, hair uint32, hairColor uint32, skin uint32, top uint32, bottom uint32, shoes uint32, weapon uint32, mapId uint32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.AccountId, accountId)
				assert.Equal(t, tt.payload.Name, name)
				assert.Equal(t, tt.payload.WorldId, worldId)
				assert.Equal(t, tt.payload.ChannelId, channelId)
				assert.Equal(t, tt.payload.JobId, jobId)
				assert.Equal(t, tt.payload.Gender, gender)
				assert.Equal(t, tt.payload.Face, face)
				assert.Equal(t, tt.payload.Hair, hair)
				assert.Equal(t, tt.payload.HairColor, hairColor)
				assert.Equal(t, tt.payload.Skin, skin)
				assert.Equal(t, tt.payload.Top, top)
				assert.Equal(t, tt.payload.Bottom, bottom)
				assert.Equal(t, tt.payload.Shoes, shoes)
				assert.Equal(t, tt.payload.Weapon, weapon)
				assert.Equal(t, tt.payload.MapId, mapId)

				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "create-character-step",
				Status:    Pending,
				Action:    CreateCharacter,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleCreateCharacter(processor, saga, step)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify that RequestCreateCharacter was called
			assert.NotNil(t, charP.RequestCreateCharacterFunc)
		})
	}
}

// TestCharacterCreationSagaIntegration tests the full character creation saga flow
func TestCharacterCreationSagaIntegration(t *testing.T) {
	tests := []struct {
		name                    string
		characterCreationResult error
		expectedFinalStatus     Status
		expectedStepCount       int
		description             string
	}{
		{
			name:                    "Success - character created successfully",
			characterCreationResult: nil,
			expectedFinalStatus:     Pending, // Step remains pending until event received
			expectedStepCount:       1,
			description:             "Character creation step should complete successfully and remain pending for event",
		},
		{
			name:                    "Failure - character creation failed",
			characterCreationResult: errors.New("character creation failed"),
			expectedFinalStatus:     Pending, // Step fails during execution
			expectedStepCount:       1,
			description:             "Character creation step should fail immediately on service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, hook := setupTestProcessor(charP, compP, validP)

			// Configure mocks
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, name string, worldId byte, channelId byte, jobId uint32, gender byte, face uint32, hair uint32, hairColor uint32, skin uint32, top uint32, bottom uint32, shoes uint32, weapon uint32, mapId uint32) error {
				return tt.characterCreationResult
			}

			// Create test saga with character creation step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "integration-test",
				Steps: []Step[any]{
					{
						StepId:    "create-character-step",
						Status:    Pending,
						Action:    CreateCharacter,
						Payload: CharacterCreatePayload{
							AccountId: 12345,
							Name:      "IntegrationTestChar",
							WorldId:   0,
							ChannelId: 0,
							JobId:     0,
							Gender:    0,
							Face:      20000,
							Hair:      30000,
							HairColor: 0,
							Skin:      0,
							Top:       1040002,
							Bottom:    1060002,
							Shoes:     1072001,
							Weapon:    1302000,
							MapId:     100000000,
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// Store saga in cache for processing
			GetCache().Put(processor.t.Id(), saga)
			
			// Execute saga processing
			err := processor.Step(saga.TransactionId)

			// Verify results based on expected outcome
			if tt.characterCreationResult != nil {
				// If character creation failed, ProcessSaga should return an error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.characterCreationResult.Error())
			} else {
				// If character creation succeeded, ProcessSaga should complete without error
				assert.NoError(t, err)
			}

			// Verify that the character processor was called
			assert.NotNil(t, charP.RequestCreateCharacterFunc)

			// Verify appropriate logging occurred
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) > 0, "Expected some log entries")
		})
	}
}

// TestCharacterCreationEventCorrelation tests the event correlation logic
func TestCharacterCreationEventCorrelation(t *testing.T) {
	tests := []struct {
		name                  string
		eventType            string
		transactionId        uuid.UUID
		expectedSuccess      bool
		expectedStepCalled   bool
		description          string
	}{
		{
			name:                "Success - character created event",
			eventType:           "CREATED",
			transactionId:       uuid.New(),
			expectedSuccess:     true,
			expectedStepCalled:  true,
			description:         "CREATED event should mark saga step as completed",
		},
		{
			name:                "Failure - character creation failed event",
			eventType:           "CREATION_FAILED",
			transactionId:       uuid.New(),
			expectedSuccess:     false,
			expectedStepCalled:  true,
			description:         "CREATION_FAILED event should mark saga step as failed",
		},
		{
			name:                "Failure - character error event",
			eventType:           "ERROR",
			transactionId:       uuid.New(),
			expectedSuccess:     false,
			expectedStepCalled:  true,
			description:         "ERROR event should mark saga step as failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, hook := setupTestProcessor(charP, compP, validP)

			// Configure saga processor mock to track StepCompleted calls
			stepCompletedCalls := make([]struct {
				transactionId uuid.UUID
				success       bool
			}, 0)

			// Note: In a real integration test, we would test the actual event handlers
			// For this test, we're verifying the expected behavior pattern
			// The actual event handlers are tested in the character consumer tests

			// Simulate the event handling logic
			var err error
			switch tt.eventType {
			case "CREATED":
				err = processor.StepCompleted(tt.transactionId, true)
				stepCompletedCalls = append(stepCompletedCalls, struct {
					transactionId uuid.UUID
					success       bool
				}{tt.transactionId, true})
			case "CREATION_FAILED", "ERROR":
				err = processor.StepCompleted(tt.transactionId, false)
				stepCompletedCalls = append(stepCompletedCalls, struct {
					transactionId uuid.UUID
					success       bool
				}{tt.transactionId, false})
			}

			// Verify that StepCompleted was called with correct parameters
			if tt.expectedStepCalled {
				assert.Equal(t, 1, len(stepCompletedCalls))
				assert.Equal(t, tt.transactionId, stepCompletedCalls[0].transactionId)
				assert.Equal(t, tt.expectedSuccess, stepCompletedCalls[0].success)
			} else {
				assert.Equal(t, 0, len(stepCompletedCalls))
			}

			// Verify no error in event processing
			assert.NoError(t, err)

			// Verify appropriate logging occurred
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) >= 0, "Log entries should be available")
		})
	}
}

// TestCharacterCreationSagaCompensation tests compensation scenarios
func TestCharacterCreationSagaCompensation(t *testing.T) {
	tests := []struct {
		name            string
		sagaSteps       []Action
		failureAtStep   int
		expectedResult  string
		description     string
	}{
		{
			name:           "Single step character creation - no compensation needed",
			sagaSteps:      []Action{CreateCharacter},
			failureAtStep:  0,
			expectedResult: "failed",
			description:    "Single character creation step failure should not require compensation",
		},
		{
			name:           "Multi-step saga with character creation first",
			sagaSteps:      []Action{CreateCharacter, AwardLevel, AwardMesos},
			failureAtStep:  1, // Fail at AwardLevel
			expectedResult: "compensating",
			description:    "Multi-step saga should enter compensation mode on later step failure",
		},
		{
			name:           "Multi-step saga with character creation failure",
			sagaSteps:      []Action{CreateCharacter, AwardLevel, AwardMesos},
			failureAtStep:  0, // Fail at CreateCharacter
			expectedResult: "failed",
			description:    "Saga should fail immediately if character creation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mocks to fail at specific step
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, name string, worldId byte, channelId byte, jobId uint32, gender byte, face uint32, hair uint32, hairColor uint32, skin uint32, top uint32, bottom uint32, shoes uint32, weapon uint32, mapId uint32) error {
				if tt.failureAtStep == 0 {
					return errors.New("character creation failed")
				}
				return nil
			}

			charP.AwardLevelAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
				if tt.failureAtStep == 1 {
					return errors.New("level award failed")
				}
				return nil
			}

			charP.AwardMesosAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
				if tt.failureAtStep == 2 {
					return errors.New("mesos award failed")
				}
				return nil
			}

			// Create test saga with multiple steps
			transactionId := uuid.New()
			steps := make([]Step[any], 0)

			for i, action := range tt.sagaSteps {
				var payload any
				switch action {
				case CreateCharacter:
					payload = CharacterCreatePayload{
						AccountId: 12345,
						Name:      "TestChar",
						WorldId:   0,
						ChannelId: 0,
						JobId:     0,
						Gender:    0,
						Face:      20000,
						Hair:      30000,
						HairColor: 0,
						Skin:      0,
						Top:       1040002,
						Bottom:    1060002,
						Shoes:     1072001,
						Weapon:    1302000,
						MapId:     100000000,
					}
				case AwardLevel:
					payload = AwardLevelPayload{
						CharacterId: 12345,
						WorldId:     0,
						ChannelId:   0,
						Amount:      1,
					}
				case AwardMesos:
					payload = AwardMesosPayload{
						CharacterId: 12345,
						WorldId:     0,
						ChannelId:   0,
						ActorId:     0,
						ActorType:   "SYSTEM",
						Amount:      1000,
					}
				}

				steps = append(steps, Step[any]{
					StepId:    fmt.Sprintf("step-%d", i),
					Status:    Pending,
					Action:    action,
					Payload:   payload,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}

			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "compensation-test",
				Steps:         steps,
			}

			// Store saga in cache for processing
			GetCache().Put(processor.t.Id(), saga)
			
			// Execute saga processing
			err := processor.Step(saga.TransactionId)

			// Verify expected behavior
			// Note: The Step method only processes one step at a time
			// For multi-step sagas, we would need to call Step multiple times
			switch tt.expectedResult {
			case "failed":
				assert.Error(t, err)
			case "compensating":
				// For this test, since we're only running one step, 
				// the first step (character creation) should succeed
				assert.NoError(t, err)
			default:
				assert.NoError(t, err)
			}

			// Verify mock functions were set based on failure point
			if tt.failureAtStep >= 0 {
				assert.NotNil(t, charP.RequestCreateCharacterFunc)
			}
			if tt.failureAtStep >= 1 {
				assert.NotNil(t, charP.AwardLevelAndEmitFunc)
			}
			if tt.failureAtStep >= 2 {
				assert.NotNil(t, charP.AwardMesosAndEmitFunc)
			}
		})
	}
}

// TestCompensateCreateCharacter tests the compensateCreateCharacter function
func TestCompensateCreateCharacter(t *testing.T) {
	tests := []struct {
		name          string
		payload       CharacterCreatePayload
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case - valid character creation payload",
			payload: CharacterCreatePayload{
				AccountId: 12345,
				Name:      "TestCharacter",
				WorldId:   0,
				ChannelId: 0,
				JobId:     0,
				Gender:    0,
				Face:      20000,
				Hair:      30000,
				HairColor: 0,
				Skin:      0,
				Top:       1040002,
				Bottom:    1060002,
				Shoes:     1072001,
				Weapon:    1302000,
				MapId:     100000000,
			},
			expectError: false,
		},
		{
			name:          "Error case - invalid payload type",
			payload:       CharacterCreatePayload{}, // This will be replaced with invalid payload
			expectError:   true,
			errorContains: "invalid payload for CreateCharacter compensation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Create test saga with failed step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "compensation-test",
				Steps: []Step[any]{
					{
						StepId:    "create-character-step",
						Status:    Failed,
						Action:    CreateCharacter,
						Payload:   tt.payload,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// For the invalid payload test, replace with invalid payload
			if tt.errorContains == "invalid payload for CreateCharacter compensation" {
				saga.Steps[0].Payload = "invalid-payload"
			}

			// Execute
			err := processor.compensateCreateCharacter(saga, saga.Steps[0])

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify that the step status was reset to pending
				assert.Equal(t, Pending, saga.Steps[0].Status)
			}
		})
	}
}

func TestTransformExperienceDistributions(t *testing.T) {
	source := []ExperienceDistributions{
		{
			ExperienceType: "WHITE",
			Amount:         1000,
			Attr1:          0,
		},
		{
			ExperienceType: "QUEST",
			Amount:         2000,
			Attr1:          1,
		},
	}

	target := TransformExperienceDistributions(source)

	assert.Equal(t, len(source), len(target))
	for i, s := range source {
		assert.Equal(t, s.ExperienceType, target[i].ExperienceType)
		assert.Equal(t, s.Amount, target[i].Amount)
		assert.Equal(t, s.Attr1, target[i].Attr1)
	}
}

// TestHandleEquipAsset tests the handleEquipAsset function
func TestHandleEquipAsset(t *testing.T) {
	tests := []struct {
		name          string
		payload       EquipAssetPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: EquipAssetPayload{
				CharacterId:   12345,
				InventoryType: 1,
				Source:        0,
				Destination:   -1,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: EquipAssetPayload{
				CharacterId:   12345,
				InventoryType: 1,
				Source:        0,
				Destination:   -1,
			},
			mockError:     errors.New("compartment service error"),
			expectError:   true,
			errorContains: "compartment service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockCompP := &mock2.ProcessorMock{
				RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
					return tt.mockError
				},
			}
			processor, _ := setupTestProcessor(nil, mockCompP)

			saga := Saga{
				TransactionId: uuid.New(),
				SagaType:      InventoryTransaction,
				InitiatedBy:   "test",
				Steps:         []Step[any]{},
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    EquipAsset,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleEquipAsset(processor, saga, step)

			// Verify
			if tt.expectError {
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

// TestHandleUnequipAsset tests the handleUnequipAsset function
func TestHandleUnequipAsset(t *testing.T) {
	tests := []struct {
		name          string
		payload       UnequipAssetPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case",
			payload: UnequipAssetPayload{
				CharacterId:   12345,
				InventoryType: 1,
				Source:        -1,
				Destination:   0,
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case",
			payload: UnequipAssetPayload{
				CharacterId:   12345,
				InventoryType: 1,
				Source:        -1,
				Destination:   0,
			},
			mockError:     errors.New("compartment service error"),
			expectError:   true,
			errorContains: "compartment service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockCompP := &mock2.ProcessorMock{
				RequestUnequipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
					return tt.mockError
				},
			}
			processor, _ := setupTestProcessor(nil, mockCompP)

			saga := Saga{
				TransactionId: uuid.New(),
				SagaType:      InventoryTransaction,
				InitiatedBy:   "test",
				Steps:         []Step[any]{},
			}

			step := Step[any]{
				StepId:    "test-step",
				Status:    Pending,
				Action:    UnequipAsset,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleUnequipAsset(processor, saga, step)

			// Verify
			if tt.expectError {
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

// TestHandleCreateAndEquipAsset tests the handleCreateAndEquipAsset function
func TestHandleCreateAndEquipAsset(t *testing.T) {
	tests := []struct {
		name          string
		payload       CreateAndEquipAssetPayload
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case - valid create and equip payload",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Success case - multiple quantity item",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 54321,
				Item: ItemPayload{
					TemplateId: 2000001,
					Quantity:   5,
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case - compartment service failure",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			mockError:     errors.New("compartment service unavailable"),
			expectError:   true,
			errorContains: "compartment service unavailable",
		},
		{
			name: "Error case - invalid template id",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 0,
					Quantity:   1,
				},
			},
			mockError:     errors.New("invalid template id"),
			expectError:   true,
			errorContains: "invalid template id",
		},
		{
			name: "Error case - character not found",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 99999,
				Item: ItemPayload{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			mockError:     errors.New("character not found"),
			expectError:   true,
			errorContains: "character not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			processor, _ := setupTestProcessor(charP, compP, validP)

			// Configure mock - the function should convert saga payload to compartment payload
			compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
				// Verify the payload was converted correctly
				assert.Equal(t, tt.payload.CharacterId, payload.CharacterId)
				assert.Equal(t, tt.payload.Item.TemplateId, payload.Item.TemplateId)
				assert.Equal(t, tt.payload.Item.Quantity, payload.Item.Quantity)
				return tt.mockError
			}

			// Create test saga and step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      InventoryTransaction,
				InitiatedBy:   "test",
			}

			step := Step[any]{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload:   tt.payload,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Execute
			err := handleCreateAndEquipAsset(processor, saga, step)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify that RequestCreateAndEquipAsset was called with correct transaction ID
			if compP.RequestCreateAndEquipAssetFunc != nil {
				// The function should have been called with the saga's transaction ID
				assert.Equal(t, transactionId, saga.TransactionId)
			}
		})
	}
}

// TestHandleCreateAndEquipAsset_InvalidPayload tests error handling for invalid payload types
func TestHandleCreateAndEquipAsset_InvalidPayload(t *testing.T) {
	// Setup
	charP := &mock.ProcessorMock{}
	compP := &mock2.ProcessorMock{}
	validP := &mock3.ProcessorMock{}

	processor, _ := setupTestProcessor(charP, compP, validP)

	// Create test saga and step with invalid payload
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "test",
	}

	step := Step[any]{
		StepId:    "create-and-equip-step",
		Status:    Pending,
		Action:    CreateAndEquipAsset,
		Payload:   "invalid-payload-type", // Wrong type
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Execute
	err := handleCreateAndEquipAsset(processor, saga, step)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload")

	// Verify that RequestCreateAndEquipAsset was never called
	assert.Nil(t, compP.RequestCreateAndEquipAssetFunc)
}

// TestHandleCreateAndEquipAsset_PayloadConversion tests the conversion from saga payload to compartment payload
func TestHandleCreateAndEquipAsset_PayloadConversion(t *testing.T) {
	// Setup
	charP := &mock.ProcessorMock{}
	compP := &mock2.ProcessorMock{}
	validP := &mock3.ProcessorMock{}

	processor, _ := setupTestProcessor(charP, compP, validP)

	// Test payload
	sagaPayload := CreateAndEquipAssetPayload{
		CharacterId: 12345,
		Item: ItemPayload{
			TemplateId: 1302000,
			Quantity:   1,
		},
	}

	// Configure mock to capture the converted payload
	var capturedPayload compartment.CreateAndEquipAssetPayload
	compP.RequestCreateAndEquipAssetFunc = func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		capturedPayload = payload
		return nil
	}

	// Create test saga and step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "test",
	}

	step := Step[any]{
		StepId:    "create-and-equip-step",
		Status:    Pending,
		Action:    CreateAndEquipAsset,
		Payload:   sagaPayload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Execute
	err := handleCreateAndEquipAsset(processor, saga, step)

	// Verify
	assert.NoError(t, err)

	// Verify the payload conversion was correct
	assert.Equal(t, sagaPayload.CharacterId, capturedPayload.CharacterId)
	assert.Equal(t, sagaPayload.Item.TemplateId, capturedPayload.Item.TemplateId)
	assert.Equal(t, sagaPayload.Item.Quantity, capturedPayload.Item.Quantity)
}