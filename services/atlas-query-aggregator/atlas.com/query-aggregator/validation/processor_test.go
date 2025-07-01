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

// TestProcessorValidate tests the Validate function of the validation processor
func TestProcessorValidate(t *testing.T) {
	// Create a logger
	logger := logrus.New()

	// Test cases
	tests := []struct {
		name              string
		characterId       uint32
		conditions        []string
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
			conditions:  []string{"jobId=100", "meso>=10000", "mapId=2000", "fame>=50"},
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
			conditions:  []string{"jobId=100", "meso>=20000", "mapId=2000", "fame>=60"},
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
			conditions:  []string{"jobId=100"},
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
			conditions:  []string{"invalid=100"},
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
			conditions:  []string{"item[2000001]>=10"},
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
					// This is a simplified version of what we did in model_test.go
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
		{
			name:        "Item condition - fail",
			characterId: 123,
			conditions:  []string{"item[2000001]>=20"},
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
					// This is a simplified version of what we did in model_test.go
					return character.NewModelBuilder().
						SetId(m.Id()).
						SetInventory(createTestInventory(m.Id())).
						Build()
				}
			},
			wantPassed:       false,
			wantDetailsCount: 1,
			wantError:        false,
		},
		{
			name:        "With decorator - add custom detail",
			characterId: 123,
			conditions:  []string{"jobId=100"},
			decorators: []model.Decorator[ValidationResult]{
				func(vr ValidationResult) ValidationResult {
					// Create a new validation result with the same character ID
					result := NewValidationResult(vr.CharacterId())

					// Add all the original details using AddResult
					// This is a workaround since we can't directly access the private fields
					// We'll add them as "passed" conditions with the original detail text
					for _, detail := range vr.Details() {
						// Extract the description part after the "Passed: " or "Failed: " prefix
						description := ""
						if strings.HasPrefix(detail, "Passed: ") {
							description = strings.TrimPrefix(detail, "Passed: ")
							result.AddResult(true, description)
						} else if strings.HasPrefix(detail, "Failed: ") {
							description = strings.TrimPrefix(detail, "Failed: ")
							result.AddResult(false, description)
						} else {
							// If there's no prefix, just add it as is
							result.AddResult(true, detail)
						}
					}

					// Add a custom detail
					result.AddResult(true, "Custom detail from decorator")
					return result
				},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							Build(), nil
					}
				}
			},
			wantPassed:       true,
			wantDetailsCount: 2, // 1 from condition + 1 from decorator
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

			// Call the Validate function with decorators
			result, err := processor.Validate(tt.decorators...)(tt.characterId, tt.conditions)

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
		})
	}
}

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

func TestValidateConditions(t *testing.T) {
	tests := []struct {
		name           string
		characterModel character.Model
		conditions     []string
		wantPassed     bool
		wantDetails    int
		wantError      bool
	}{
		{
			name: "All conditions pass",
			characterModel: character.NewModelBuilder().
				SetJobId(100).
				SetMeso(10000).
				SetMapId(2000).
				SetFame(50).
				Build(),
			conditions:  []string{"jobId=100", "meso>=10000", "mapId=2000", "fame>=50"},
			wantPassed:  true,
			wantDetails: 4,
			wantError:   false,
		},
		{
			name: "Some conditions fail",
			characterModel: character.NewModelBuilder().
				SetJobId(100).
				SetMeso(10000).
				SetMapId(2000).
				SetFame(50).
				Build(),
			conditions:  []string{"jobId=100", "meso>=20000", "mapId=2000", "fame>=60"},
			wantPassed:  false,
			wantDetails: 4,
			wantError:   false,
		},
		{
			name: "Item condition - pass",
			characterModel: character.NewModelBuilder().
				SetId(123).
				SetInventory(createTestInventory(123)).
				Build(),
			conditions:  []string{"item[2000001]>=10"},
			wantPassed:  true,
			wantDetails: 1,
			wantError:   false,
		},
		{
			name: "Item condition - fail",
			characterModel: character.NewModelBuilder().
				SetId(123).
				SetInventory(createTestInventory(123)).
				Build(),
			conditions:  []string{"item[2000001]>=20"},
			wantPassed:  false,
			wantDetails: 1,
			wantError:   false,
		},
		{
			name: "Item not found",
			characterModel: character.NewModelBuilder().
				SetId(123).
				SetInventory(createTestInventory(123)).
				Build(),
			conditions:  []string{"item[9999999]=10"},
			wantPassed:  false,
			wantDetails: 1,
			wantError:   false,
		},
		{
			name: "Mixed conditions",
			characterModel: character.NewModelBuilder().
				SetId(123).
				SetJobId(100).
				SetMeso(10000).
				SetMapId(2000).
				SetFame(50).
				SetInventory(createTestInventory(123)).
				Build(),
			conditions:  []string{"jobId=100", "meso>=10000", "item[2000001]>=10", "item[2000002]>=10"},
			wantPassed:  false, // One item condition fails
			wantDetails: 4,
			wantError:   false,
		},
		{
			name:           "Invalid condition",
			characterModel: character.NewModelBuilder().Build(),
			conditions:     []string{"invalid=100"},
			wantPassed:     true, // Default is true, but error will be returned
			wantDetails:    0,
			wantError:      true,
		},
		{
			name:           "Empty conditions",
			characterModel: character.NewModelBuilder().Build(),
			conditions:     []string{},
			wantPassed:     true, // No conditions means all pass
			wantDetails:    0,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a validation result
			result := NewValidationResult(123)

			var err error
			// Process each condition
			for _, expr := range tt.conditions {
				var condition Condition
				condition, err = NewCondition(expr)
				if err != nil {
					break
				}

				// Evaluate the condition
				passed, description := condition.Evaluate(tt.characterModel)
				result.AddResult(passed, description)
			}

			// Check for expected errors
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
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

			if len(result.Details()) != tt.wantDetails {
				t.Errorf("Validation details count = %v, want %v", len(result.Details()), tt.wantDetails)
			}
		})
	}
}
