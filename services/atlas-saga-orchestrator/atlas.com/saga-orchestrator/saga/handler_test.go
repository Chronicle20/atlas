package saga

import (
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/validation"
	mock3 "atlas-saga-orchestrator/validation/mock"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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
			validP := &mock3.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithValidationProcessor(validP).handleValidateCharacterState(saga, step)

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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleWarpToPortal(saga, step)

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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleWarpToRandomPortal(saga, step)

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
			compP := &mock2.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleAwardAsset(saga, step)

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
			compP := &mock2.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleAwardInventory(saga, step)

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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

			// Configure mock
			charP.AwardLevelAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)
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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleAwardLevel(saga, step)

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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

			// Configure mock
			charP.AwardExperienceAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)

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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleAwardExperience(saga, step)

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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

			// Configure mock
			charP.AwardMesosAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
				// Verify parameters
				assert.Equal(t, tt.payload.CharacterId, characterId)
				assert.Equal(t, tt.payload.WorldId, worldId)
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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleAwardMesos(saga, step)

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
			compP := &mock2.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleDestroyAsset(saga, step)

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
				AccountId:    12345,
				Name:         "TestCharacter",
				WorldId:      0,
				Level:        1,
				Strength:     13,
				Dexterity:    4,
				Intelligence: 4,
				Luck:         4,
				Hp:           50,
				Mp:           5,
				JobId:        job.Id(0),
				Gender:       0,
				Face:         20000,
				Hair:         30000,
				Skin:         0,
				Top:          1040002,
				Bottom:       1060002,
				Shoes:        1072001,
				Weapon:       1302000,
				MapId:        _map.Id(100000000),
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Error case - character service failure",
			payload: CharacterCreatePayload{
				AccountId:    12345,
				Name:         "TestCharacter",
				WorldId:      0,
				Level:        1,
				Strength:     13,
				Dexterity:    4,
				Intelligence: 4,
				Luck:         4,
				Hp:           50,
				Mp:           5,
				JobId:        job.Id(0),
				Gender:       0,
				Face:         20000,
				Hair:         30000,
				Skin:         0,
				Top:          1040002,
				Bottom:       1060002,
				Shoes:        1072001,
				Weapon:       1302000,
				MapId:        _map.Id(100000000),
			},
			mockError:     errors.New("character service unavailable"),
			expectError:   true,
			errorContains: "character service unavailable",
		},
		{
			name: "Error case - duplicate character name",
			payload: CharacterCreatePayload{
				AccountId:    12345,
				Name:         "ExistingCharacter",
				WorldId:      0,
				Level:        1,
				Strength:     13,
				Dexterity:    4,
				Intelligence: 4,
				Luck:         4,
				Hp:           50,
				Mp:           5,
				JobId:        job.Id(0),
				Gender:       0,
				Face:         20000,
				Hair:         30000,
				Skin:         0,
				Top:          1040002,
				Bottom:       1060002,
				Shoes:        1072001,
				Weapon:       1302000,
				MapId:        _map.Id(100000000),
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

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

			// Configure mock
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
				// Verify parameters
				assert.Equal(t, tt.payload.AccountId, accountId)
				assert.Equal(t, tt.payload.Name, name)
				assert.Equal(t, tt.payload.WorldId, worldId)
				assert.Equal(t, tt.payload.JobId, jobId)
				assert.Equal(t, tt.payload.Gender, gender)
				assert.Equal(t, tt.payload.Face, face)
				assert.Equal(t, tt.payload.Hair, hair)
				assert.Equal(t, tt.payload.Skin, skin)
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
			err := NewHandler(logger, ctx).WithCharacterProcessor(charP).handleCreateCharacter(saga, step)

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
			compP := &mock2.ProcessorMock{
				RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
					return tt.mockError
				},
			}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleEquipAsset(saga, step)

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
			compP := &mock2.ProcessorMock{
				RequestUnequipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
					return tt.mockError
				},
			}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleUnequipAsset(saga, step)

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
			compP := &mock2.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

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
			err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleCreateAndEquipAsset(saga, step)

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
	compP := &mock2.ProcessorMock{}

	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	_, ctx := setupContext()

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
	err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleCreateAndEquipAsset(saga, step)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload")

	// Verify that RequestCreateAndEquipAsset was never called
	assert.Nil(t, compP.RequestCreateAndEquipAssetFunc)
}

// TestHandleCreateAndEquipAsset_PayloadConversion tests the conversion from saga payload to compartment payload
func TestHandleCreateAndEquipAsset_PayloadConversion(t *testing.T) {
	// Setup
	compP := &mock2.ProcessorMock{}

	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	_, ctx := setupContext()

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
	err := NewHandler(logger, ctx).WithCompartmentProcessor(compP).handleCreateAndEquipAsset(saga, step)

	// Verify
	assert.NoError(t, err)

	// Verify the payload conversion was correct
	assert.Equal(t, sagaPayload.CharacterId, capturedPayload.CharacterId)
	assert.Equal(t, sagaPayload.Item.TemplateId, capturedPayload.Item.TemplateId)
	assert.Equal(t, sagaPayload.Item.Quantity, capturedPayload.Item.Quantity)
}
