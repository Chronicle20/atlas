package validation

import (
	"atlas-query-aggregator/asset"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/character/mock"
	"atlas-query-aggregator/compartment"
	"atlas-query-aggregator/inventory"
	"context"
	"errors"
	inventory_type "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
	"time"
)


// TestValidateConditions tests the condition validation logic directly
// Helper function to create a test inventory with items for processor tests
func createTestInventory(characterId uint32) inventory.Model {
	// Create a test compartment with items
	compartmentId := uuid.New()

	// Create a builder
	builder := compartment.NewBuilder(compartmentId, characterId, inventory_type.TypeValueUse, 100)

	// Add some test items
	// Item 2000001 with quantity 10
	refData1 := asset.NewConsumableReferenceDataBuilder().
		SetQuantity(10).
		Build()
	item1 := asset.NewBuilder[any](1, compartmentId, 2000001, 1, asset.ReferenceTypeConsumable).
		SetSlot(1).
		SetExpiration(time.Now().Add(24 * time.Hour)).
		SetReferenceData(refData1).
		Build()
	builder.AddAsset(item1)

	// Item 2000002 with quantity 5
	refData2 := asset.NewConsumableReferenceDataBuilder().
		SetQuantity(5).
		Build()
	item2 := asset.NewBuilder[any](2, compartmentId, 2000002, 2, asset.ReferenceTypeConsumable).
		SetSlot(2).
		SetExpiration(time.Now().Add(24 * time.Hour)).
		SetReferenceData(refData2).
		Build()
	builder.AddAsset(item2)

	// Item 2000003 with quantity 20
	refData3 := asset.NewConsumableReferenceDataBuilder().
		SetQuantity(20).
		Build()
	item3 := asset.NewBuilder[any](3, compartmentId, 2000003, 3, asset.ReferenceTypeConsumable).
		SetSlot(3).
		SetExpiration(time.Now().Add(24 * time.Hour)).
		SetReferenceData(refData3).
		Build()
	builder.AddAsset(item3)

	// Create inventory model
	return inventory.NewBuilder(characterId).
		SetConsumable(builder.Build()).
		Build()
}

// TestConditionBuilder tests the ConditionBuilder
func TestConditionBuilder(t *testing.T) {
	tests := []struct {
		name        string
		input       ConditionInput
		wantType    ConditionType
		wantOp      Operator
		wantValue   int
		wantItemId  uint32
		shouldError bool
	}{
		{
			name: "Valid job equals condition",
			input: ConditionInput{
				Type:     "jobId",
				Operator: "=",
				Value:    100,
			},
			wantType:    JobCondition,
			wantOp:      Equals,
			wantValue:   100,
			shouldError: false,
		},
		{
			name: "Valid meso greater than condition",
			input: ConditionInput{
				Type:     "meso",
				Operator: ">",
				Value:    10000,
			},
			wantType:    MesoCondition,
			wantOp:      GreaterThan,
			wantValue:   10000,
			shouldError: false,
		},
		{
			name: "Valid item equals condition",
			input: ConditionInput{
				Type:     "item",
				Operator: "=",
				Value:    10,
				ItemId:   2000001,
			},
			wantType:    ItemCondition,
			wantOp:      Equals,
			wantValue:   10,
			wantItemId:  2000001,
			shouldError: false,
		},
		{
			name: "Invalid condition type",
			input: ConditionInput{
				Type:     "invalid",
				Operator: "=",
				Value:    100,
			},
			shouldError: true,
		},
		{
			name: "Invalid operator",
			input: ConditionInput{
				Type:     "jobId",
				Operator: "invalid",
				Value:    100,
			},
			shouldError: true,
		},
		{
			name: "Item condition without ItemId",
			input: ConditionInput{
				Type:     "item",
				Operator: "=",
				Value:    10,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewConditionBuilder()
			condition, err := builder.FromInput(tt.input).Build()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if condition.conditionType != tt.wantType {
				t.Errorf("ConditionBuilder.Build() conditionType = %v, want %v", condition.conditionType, tt.wantType)
			}

			if condition.operator != tt.wantOp {
				t.Errorf("ConditionBuilder.Build() operator = %v, want %v", condition.operator, tt.wantOp)
			}

			if condition.value != tt.wantValue {
				t.Errorf("ConditionBuilder.Build() value = %v, want %v", condition.value, tt.wantValue)
			}

			if tt.wantType == ItemCondition && condition.referenceId != tt.wantItemId {
				t.Errorf("ConditionBuilder.Build() itemId = %v, want %v", condition.referenceId, tt.wantItemId)
			}
		})
	}
}

// TestProcessorValidateStructured tests the ValidateStructured function of the validation processor
func TestProcessorValidateStructured(t *testing.T) {
	// Create a logger
	logger := logrus.New()

	// Test cases
	tests := []struct {
		name              string
		characterId       uint32
		conditions        []ConditionInput
		decorators        []model.Decorator[ValidationResult]
		setupMock         func(*mock.ProcessorImpl)
		wantPassed        bool
		wantDetailsCount  int
		wantError         bool
		wantErrorContains string
	}{
		{
			name:        "All conditions pass",
			characterId: 123,
			conditions: []ConditionInput{
				{Type: "jobId", Operator: "=", Value: 100},
				{Type: "meso", Operator: ">=", Value: 10000},
				{Type: "mapId", Operator: "=", Value: 2000},
				{Type: "fame", Operator: ">=", Value: 50},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							SetMeso(10000).
							SetMapId(2000).
							SetFame(50).
							Build(), nil
					}
				}
			},
			wantPassed:       true,
			wantDetailsCount: 4,
			wantError:        false,
		},
		{
			name:        "Some conditions fail",
			characterId: 123,
			conditions: []ConditionInput{
				{Type: "jobId", Operator: "=", Value: 100},
				{Type: "meso", Operator: ">=", Value: 20000},
				{Type: "mapId", Operator: "=", Value: 2000},
				{Type: "fame", Operator: ">=", Value: 60},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							SetMeso(10000).
							SetMapId(2000).
							SetFame(50).
							Build(), nil
					}
				}
			},
			wantPassed:       false,
			wantDetailsCount: 4,
			wantError:        false,
		},
		{
			name:        "Error getting character data",
			characterId: 123,
			conditions: []ConditionInput{
				{Type: "jobId", Operator: "=", Value: 100},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.Model{}, errors.New("character not found")
					}
				}
			},
			wantPassed:        false,
			wantDetailsCount:  0,
			wantError:         true,
			wantErrorContains: "failed to get character data",
		},
		{
			name:        "Invalid condition",
			characterId: 123,
			conditions: []ConditionInput{
				{Type: "invalid", Operator: "=", Value: 100},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().Build(), nil
					}
				}
			},
			wantPassed:        false,
			wantDetailsCount:  0,
			wantError:         true,
			wantErrorContains: "invalid condition",
		},
		{
			name:        "Item condition - pass",
			characterId: 123,
			conditions: []ConditionInput{
				{Type: "item", Operator: ">=", Value: 10, ItemId: 2000001},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					// If a decorator is provided (InventoryDecorator), apply it to the model
					if len(decorators) > 0 {
						return func(characterId uint32) (character.Model, error) {
							// Create a basic character
							char := character.NewModelBuilder().
								SetId(characterId).
								Build()

							// Apply the decorator (which should be InventoryDecorator)
							return decorators[0](char), nil
						}
					}

					// Otherwise return a basic character
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetId(characterId).
							Build(), nil
					}
				}

				// Mock the InventoryDecorator to add inventory with items
				m.InventoryDecoratorFunc = func(m character.Model) character.Model {
					// Create a test inventory with items
					return character.NewModelBuilder().
						SetId(m.Id()).
						SetInventory(createTestInventory(m.Id())).
						Build()
				}
			},
			wantPassed:       true,
			wantDetailsCount: 1,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock character processor
			mockCharProcessor := &mock.ProcessorImpl{}
			if tt.setupMock != nil {
				tt.setupMock(mockCharProcessor)
			}

			// Create a validation processor with the mock character processor
			processor := &ProcessorImpl{
				l:                  logger,
				ctx:                context.Background(),
				characterProcessor: mockCharProcessor,
			}

			// Call the ValidateStructured function with decorators
			result, err := processor.ValidateStructured(tt.decorators...)(tt.characterId, tt.conditions)

			// Check for expected errors
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.wantErrorContains != "" && !strings.Contains(err.Error(), tt.wantErrorContains) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.wantErrorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check validation result
			if result.Passed() != tt.wantPassed {
				t.Errorf("Validation passed = %v, want %v", result.Passed(), tt.wantPassed)
			}

			if len(result.Details()) != tt.wantDetailsCount {
				t.Errorf("Validation details count = %v, want %v", len(result.Details()), tt.wantDetailsCount)
			}

			// Check that results field is populated
			if len(result.Results()) != tt.wantDetailsCount {
				t.Errorf("Validation results count = %v, want %v", len(result.Results()), tt.wantDetailsCount)
			}
		})
	}
}

