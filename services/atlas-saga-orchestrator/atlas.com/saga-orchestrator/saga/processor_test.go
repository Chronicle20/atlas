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
