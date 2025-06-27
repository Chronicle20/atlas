package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	character2 "atlas-saga-orchestrator/kafka/message/character"
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
func setupTestProcessor(charP character.Processor, compP compartment.Processor) (*ProcessorImpl, *test.Hook) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	processor := NewProcessor(logger, tctx)
	processor.charP = charP
	processor.compP = compP
	return processor, hook
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

			processor, _ := setupTestProcessor(charP, compP)

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

			processor, _ := setupTestProcessor(charP, compP)

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

			processor, _ := setupTestProcessor(charP, compP)

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
				SagaType:      InventoryTransaction,
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
				Amount:      2,
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

			processor, _ := setupTestProcessor(charP, compP)

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

			processor, _ := setupTestProcessor(charP, compP)

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

func TestTransformExperienceDistributions(t *testing.T) {
	tests := []struct {
		name     string
		source   []ExperienceDistributions
		expected []character2.ExperienceDistributions
	}{
		{
			name:     "Empty slice",
			source:   []ExperienceDistributions{},
			expected: []character2.ExperienceDistributions{},
		},
		{
			name: "Single distribution",
			source: []ExperienceDistributions{
				{
					ExperienceType: "WHITE",
					Amount:         1000,
					Attr1:          0,
				},
			},
			expected: []character2.ExperienceDistributions{
				{
					ExperienceType: "WHITE",
					Amount:         1000,
					Attr1:          0,
				},
			},
		},
		{
			name: "Multiple distributions",
			source: []ExperienceDistributions{
				{
					ExperienceType: "WHITE",
					Amount:         1000,
					Attr1:          0,
				},
				{
					ExperienceType: "YELLOW",
					Amount:         2000,
					Attr1:          5,
				},
				{
					ExperienceType: "PARTY",
					Amount:         3000,
					Attr1:          10,
				},
			},
			expected: []character2.ExperienceDistributions{
				{
					ExperienceType: "WHITE",
					Amount:         1000,
					Attr1:          0,
				},
				{
					ExperienceType: "YELLOW",
					Amount:         2000,
					Attr1:          5,
				},
				{
					ExperienceType: "PARTY",
					Amount:         3000,
					Attr1:          10,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformExperienceDistributions(tt.source)
			assert.Equal(t, tt.expected, result)
		})
	}
}
