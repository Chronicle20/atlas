package validation

import (
	"atlas-query-aggregator/asset"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/compartment"
	"atlas-query-aggregator/inventory"
	inventory_type "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
	"testing"
	"time"
)


func TestCondition_Evaluate(t *testing.T) {
	// Create test inventory with items
	compartmentId := uuid.New()
	consumableCompartment := createTestCompartment(compartmentId, 123, inventory_type.TypeValueUse, 100)

	// Create inventory model
	inventoryModel := inventory.NewBuilder(123).
		SetConsumable(consumableCompartment).
		Build()

	// Create a test character with inventory
	character := character.NewModelBuilder().
		SetId(123).
		SetJobId(100).
		SetMeso(10000).
		SetMapId(2000).
		SetFame(50).
		SetGender(0). // 0 = male
		SetLevel(25).
		SetReborns(3).
		SetDojoPoints(1500).
		SetVanquisherKills(7).
		SetStrength(50).
		SetDexterity(35).
		SetIntelligence(20).
		SetLuck(15).
		SetInventory(inventoryModel).
		Build()

	tests := []struct {
		name         string
		condition    Condition
		wantPassed   bool
		wantContains string
	}{
		// Item condition tests
		{
			name: "Item equals - pass",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         10,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity = 10",
		},
		{
			name: "Item equals - fail",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         15,
				referenceId:        2000001,
			},
			wantPassed:   false,
			wantContains: "Item 2000001 quantity = 15",
		},
		{
			name: "Item greater than - pass",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      GreaterThan,
				value:         5,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity > 5",
		},
		{
			name: "Item greater than - fail",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      GreaterThan,
				value:         15,
				referenceId:        2000001,
			},
			wantPassed:   false,
			wantContains: "Item 2000001 quantity > 15",
		},
		{
			name: "Item less than - pass",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      LessThan,
				value:         15,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity < 15",
		},
		{
			name: "Item less than - fail",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      LessThan,
				value:         5,
				referenceId:        2000001,
			},
			wantPassed:   false,
			wantContains: "Item 2000001 quantity < 5",
		},
		{
			name: "Item greater than or equal - pass (equal)",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      GreaterEqual,
				value:         10,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity >= 10",
		},
		{
			name: "Item greater than or equal - pass (greater)",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      GreaterEqual,
				value:         5,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity >= 5",
		},
		{
			name: "Item greater than or equal - fail",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      GreaterEqual,
				value:         15,
				referenceId:        2000001,
			},
			wantPassed:   false,
			wantContains: "Item 2000001 quantity >= 15",
		},
		{
			name: "Item less than or equal - pass (equal)",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      LessEqual,
				value:         10,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity <= 10",
		},
		{
			name: "Item less than or equal - pass (less)",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      LessEqual,
				value:         15,
				referenceId:        2000001,
			},
			wantPassed:   true,
			wantContains: "Item 2000001 quantity <= 15",
		},
		{
			name: "Item less than or equal - fail",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      LessEqual,
				value:         5,
				referenceId:        2000001,
			},
			wantPassed:   false,
			wantContains: "Item 2000001 quantity <= 5",
		},
		{
			name: "Item not found",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         10,
				referenceId:        9999999, // Non-existent item
			},
			wantPassed:   false,
			wantContains: "Invalid item ID: 9999999",
		},
		{
			name: "Job equals - pass",
			condition: Condition{
				conditionType: JobCondition,
				operator:      Equals,
				value:         100,
			},
			wantPassed:   true,
			wantContains: "Job ID = 100",
		},
		{
			name: "Job equals - fail",
			condition: Condition{
				conditionType: JobCondition,
				operator:      Equals,
				value:         200,
			},
			wantPassed:   false,
			wantContains: "Job ID = 200",
		},
		{
			name: "Meso greater than - pass",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      GreaterThan,
				value:         9000,
			},
			wantPassed:   true,
			wantContains: "Meso > 9000",
		},
		{
			name: "Meso greater than - fail",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      GreaterThan,
				value:         11000,
			},
			wantPassed:   false,
			wantContains: "Meso > 11000",
		},
		{
			name: "Map less than - pass",
			condition: Condition{
				conditionType: MapCondition,
				operator:      LessThan,
				value:         3000,
			},
			wantPassed:   true,
			wantContains: "Map ID < 3000",
		},
		{
			name: "Map less than - fail",
			condition: Condition{
				conditionType: MapCondition,
				operator:      LessThan,
				value:         1000,
			},
			wantPassed:   false,
			wantContains: "Map ID < 1000",
		},
		{
			name: "Fame greater than or equal - pass (equal)",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         50,
			},
			wantPassed:   true,
			wantContains: "Fame >= 50",
		},
		{
			name: "Fame greater than or equal - pass (greater)",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         40,
			},
			wantPassed:   true,
			wantContains: "Fame >= 40",
		},
		{
			name: "Fame greater than or equal - fail",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         60,
			},
			wantPassed:   false,
			wantContains: "Fame >= 60",
		},
		{
			name: "Meso less than or equal - pass (equal)",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         10000,
			},
			wantPassed:   true,
			wantContains: "Meso <= 10000",
		},
		{
			name: "Meso less than or equal - pass (less)",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         11000,
			},
			wantPassed:   true,
			wantContains: "Meso <= 11000",
		},
		{
			name: "Meso less than or equal - fail",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         9000,
			},
			wantPassed:   false,
			wantContains: "Meso <= 9000",
		},
		{
			name: "Gender equals - pass (male)",
			condition: Condition{
				conditionType: GenderCondition,
				operator:      Equals,
				value:         0, // 0 = male
			},
			wantPassed:   true,
			wantContains: "Gender = 0",
		},
		{
			name: "Gender equals - fail (female)",
			condition: Condition{
				conditionType: GenderCondition,
				operator:      Equals,
				value:         1, // 1 = female
			},
			wantPassed:   false,
			wantContains: "Gender = 1",
		},
		{
			name: "Gender not equals - pass (not female)",
			condition: Condition{
				conditionType: GenderCondition,
				operator:      GreaterThan,
				value:         0, // 0 = male, 1 = female
			},
			wantPassed:   false,
			wantContains: "Gender > 0",
		},
		{
			name: "Gender less than - pass (male < 1)",
			condition: Condition{
				conditionType: GenderCondition,
				operator:      LessThan,
				value:         1, // 0 = male, 1 = female
			},
			wantPassed:   true,
			wantContains: "Gender < 1",
		},
		// Level condition tests
		{
			name: "Level equals - pass",
			condition: Condition{
				conditionType: LevelCondition,
				operator:      Equals,
				value:         25,
			},
			wantPassed:   true,
			wantContains: "Level = 25",
		},
		{
			name: "Level equals - fail",
			condition: Condition{
				conditionType: LevelCondition,
				operator:      Equals,
				value:         30,
			},
			wantPassed:   false,
			wantContains: "Level = 30",
		},
		{
			name: "Level greater than - pass",
			condition: Condition{
				conditionType: LevelCondition,
				operator:      GreaterThan,
				value:         20,
			},
			wantPassed:   true,
			wantContains: "Level > 20",
		},
		{
			name: "Level greater than - fail",
			condition: Condition{
				conditionType: LevelCondition,
				operator:      GreaterThan,
				value:         30,
			},
			wantPassed:   false,
			wantContains: "Level > 30",
		},
		// Reborns condition tests
		{
			name: "Reborns equals - pass",
			condition: Condition{
				conditionType: RebornsCondition,
				operator:      Equals,
				value:         3,
			},
			wantPassed:   true,
			wantContains: "Reborns = 3",
		},
		{
			name: "Reborns equals - fail",
			condition: Condition{
				conditionType: RebornsCondition,
				operator:      Equals,
				value:         5,
			},
			wantPassed:   false,
			wantContains: "Reborns = 5",
		},
		{
			name: "Reborns greater than or equal - pass",
			condition: Condition{
				conditionType: RebornsCondition,
				operator:      GreaterEqual,
				value:         2,
			},
			wantPassed:   true,
			wantContains: "Reborns >= 2",
		},
		{
			name: "Reborns greater than or equal - fail",
			condition: Condition{
				conditionType: RebornsCondition,
				operator:      GreaterEqual,
				value:         5,
			},
			wantPassed:   false,
			wantContains: "Reborns >= 5",
		},
		// Dojo Points condition tests
		{
			name: "Dojo Points equals - pass",
			condition: Condition{
				conditionType: DojoPointsCondition,
				operator:      Equals,
				value:         1500,
			},
			wantPassed:   true,
			wantContains: "Dojo Points = 1500",
		},
		{
			name: "Dojo Points equals - fail",
			condition: Condition{
				conditionType: DojoPointsCondition,
				operator:      Equals,
				value:         2000,
			},
			wantPassed:   false,
			wantContains: "Dojo Points = 2000",
		},
		{
			name: "Dojo Points greater than - pass",
			condition: Condition{
				conditionType: DojoPointsCondition,
				operator:      GreaterThan,
				value:         1000,
			},
			wantPassed:   true,
			wantContains: "Dojo Points > 1000",
		},
		{
			name: "Dojo Points greater than - fail",
			condition: Condition{
				conditionType: DojoPointsCondition,
				operator:      GreaterThan,
				value:         2000,
			},
			wantPassed:   false,
			wantContains: "Dojo Points > 2000",
		},
		// Vanquisher Kills condition tests
		{
			name: "Vanquisher Kills equals - pass",
			condition: Condition{
				conditionType: VanquisherKillsCondition,
				operator:      Equals,
				value:         7,
			},
			wantPassed:   true,
			wantContains: "Vanquisher Kills = 7",
		},
		{
			name: "Vanquisher Kills equals - fail",
			condition: Condition{
				conditionType: VanquisherKillsCondition,
				operator:      Equals,
				value:         10,
			},
			wantPassed:   false,
			wantContains: "Vanquisher Kills = 10",
		},
		{
			name: "Vanquisher Kills less than - pass",
			condition: Condition{
				conditionType: VanquisherKillsCondition,
				operator:      LessThan,
				value:         10,
			},
			wantPassed:   true,
			wantContains: "Vanquisher Kills < 10",
		},
		{
			name: "Vanquisher Kills less than - fail",
			condition: Condition{
				conditionType: VanquisherKillsCondition,
				operator:      LessThan,
				value:         5,
			},
			wantPassed:   false,
			wantContains: "Vanquisher Kills < 5",
		},
		// Strength condition tests
		{
			name: "Strength equals - pass",
			condition: Condition{
				conditionType: StrengthCondition,
				operator:      Equals,
				value:         50,
			},
			wantPassed:   true,
			wantContains: "Strength = 50",
		},
		{
			name: "Strength equals - fail",
			condition: Condition{
				conditionType: StrengthCondition,
				operator:      Equals,
				value:         60,
			},
			wantPassed:   false,
			wantContains: "Strength = 60",
		},
		{
			name: "Strength greater than - pass",
			condition: Condition{
				conditionType: StrengthCondition,
				operator:      GreaterThan,
				value:         40,
			},
			wantPassed:   true,
			wantContains: "Strength > 40",
		},
		{
			name: "Strength greater than - fail",
			condition: Condition{
				conditionType: StrengthCondition,
				operator:      GreaterThan,
				value:         60,
			},
			wantPassed:   false,
			wantContains: "Strength > 60",
		},
		// Dexterity condition tests
		{
			name: "Dexterity equals - pass",
			condition: Condition{
				conditionType: DexterityCondition,
				operator:      Equals,
				value:         35,
			},
			wantPassed:   true,
			wantContains: "Dexterity = 35",
		},
		{
			name: "Dexterity equals - fail",
			condition: Condition{
				conditionType: DexterityCondition,
				operator:      Equals,
				value:         45,
			},
			wantPassed:   false,
			wantContains: "Dexterity = 45",
		},
		{
			name: "Dexterity less than or equal - pass",
			condition: Condition{
				conditionType: DexterityCondition,
				operator:      LessEqual,
				value:         35,
			},
			wantPassed:   true,
			wantContains: "Dexterity <= 35",
		},
		{
			name: "Dexterity less than or equal - fail",
			condition: Condition{
				conditionType: DexterityCondition,
				operator:      LessEqual,
				value:         30,
			},
			wantPassed:   false,
			wantContains: "Dexterity <= 30",
		},
		// Intelligence condition tests
		{
			name: "Intelligence equals - pass",
			condition: Condition{
				conditionType: IntelligenceCondition,
				operator:      Equals,
				value:         20,
			},
			wantPassed:   true,
			wantContains: "Intelligence = 20",
		},
		{
			name: "Intelligence equals - fail",
			condition: Condition{
				conditionType: IntelligenceCondition,
				operator:      Equals,
				value:         30,
			},
			wantPassed:   false,
			wantContains: "Intelligence = 30",
		},
		{
			name: "Intelligence greater than or equal - pass",
			condition: Condition{
				conditionType: IntelligenceCondition,
				operator:      GreaterEqual,
				value:         15,
			},
			wantPassed:   true,
			wantContains: "Intelligence >= 15",
		},
		{
			name: "Intelligence greater than or equal - fail",
			condition: Condition{
				conditionType: IntelligenceCondition,
				operator:      GreaterEqual,
				value:         25,
			},
			wantPassed:   false,
			wantContains: "Intelligence >= 25",
		},
		// Luck condition tests
		{
			name: "Luck equals - pass",
			condition: Condition{
				conditionType: LuckCondition,
				operator:      Equals,
				value:         15,
			},
			wantPassed:   true,
			wantContains: "Luck = 15",
		},
		{
			name: "Luck equals - fail",
			condition: Condition{
				conditionType: LuckCondition,
				operator:      Equals,
				value:         25,
			},
			wantPassed:   false,
			wantContains: "Luck = 25",
		},
		{
			name: "Luck less than - pass",
			condition: Condition{
				conditionType: LuckCondition,
				operator:      LessThan,
				value:         20,
			},
			wantPassed:   true,
			wantContains: "Luck < 20",
		},
		{
			name: "Luck less than - fail",
			condition: Condition{
				conditionType: LuckCondition,
				operator:      LessThan,
				value:         10,
			},
			wantPassed:   false,
			wantContains: "Luck < 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.Evaluate(character)

			if result.Passed != tt.wantPassed {
				t.Errorf("Condition.Evaluate() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Description != tt.wantContains {
				t.Errorf("Condition.Evaluate() description = %v, want to contain %v", result.Description, tt.wantContains)
			}
		})
	}
}

// Helper function to create a test compartment with items
func createTestCompartment(id uuid.UUID, characterId uint32, inventoryType inventory_type.Type, capacity uint32) compartment.Model {
	// Create a builder
	builder := compartment.NewBuilder(id, characterId, inventoryType, capacity)

	// Add some test items
	// Item 2000001 with quantity 10
	item1 := createTestItem(1, id, 2000001, 10)
	builder.AddAsset(item1)

	// Item 2000002 with quantity 5
	item2 := createTestItem(2, id, 2000002, 5)
	builder.AddAsset(item2)

	// Item 2000003 with quantity 20
	item3 := createTestItem(3, id, 2000003, 20)
	builder.AddAsset(item3)

	return builder.Build()
}

// Helper function to create a test item
func createTestItem(id uint32, compartmentId uuid.UUID, templateId uint32, quantity uint32) asset.Model[any] {
	// Create consumable reference data using builder
	refData := asset.NewConsumableReferenceDataBuilder().
		SetQuantity(quantity).
		Build()

	// Create the asset
	return asset.NewBuilder[any](id, compartmentId, templateId, id, asset.ReferenceTypeConsumable).
		SetSlot(int16(id)).
		SetExpiration(time.Now().Add(24 * time.Hour)).
		SetReferenceData(refData).
		Build()
}

func TestValidationResult(t *testing.T) {
	t.Run("New validation result", func(t *testing.T) {
		result := NewValidationResult(123)

		if !result.Passed() {
			t.Errorf("NewValidationResult() passed = %v, want true", result.Passed())
		}

		if len(result.Details()) != 0 {
			t.Errorf("NewValidationResult() details length = %v, want 0", len(result.Details()))
		}

		if result.CharacterId() != 123 {
			t.Errorf("NewValidationResult() characterId = %v, want 123", result.CharacterId())
		}
	})

	t.Run("Add passing condition result", func(t *testing.T) {
		result := NewValidationResult(123)
		condResult := ConditionResult{
			Passed:      true,
			Description: "Test condition",
			Type:        JobCondition,
			Operator:    Equals,
			Value:       100,
			ActualValue: 100,
		}
		result.AddConditionResult(condResult)

		if !result.Passed() {
			t.Errorf("After AddConditionResult(passing) passed = %v, want true", result.Passed())
		}

		if len(result.Details()) != 1 {
			t.Errorf("After AddConditionResult() details length = %v, want 1", len(result.Details()))
		}

		if result.Details()[0] != "Passed: Test condition" {
			t.Errorf("After AddConditionResult() detail = %v, want 'Passed: Test condition'", result.Details()[0])
		}
	})

	t.Run("Add failing condition result", func(t *testing.T) {
		result := NewValidationResult(123)
		condResult := ConditionResult{
			Passed:      false,
			Description: "Test condition",
			Type:        JobCondition,
			Operator:    Equals,
			Value:       100,
			ActualValue: 200,
		}
		result.AddConditionResult(condResult)

		if result.Passed() {
			t.Errorf("After AddConditionResult(failing) passed = %v, want false", result.Passed())
		}

		if len(result.Details()) != 1 {
			t.Errorf("After AddConditionResult() details length = %v, want 1", len(result.Details()))
		}

		if result.Details()[0] != "Failed: Test condition" {
			t.Errorf("After AddConditionResult() detail = %v, want 'Failed: Test condition'", result.Details()[0])
		}
	})

	t.Run("Multiple condition results", func(t *testing.T) {
		result := NewValidationResult(123)

		condResult1 := ConditionResult{
			Passed:      true,
			Description: "Condition 1",
			Type:        JobCondition,
			Operator:    Equals,
			Value:       100,
			ActualValue: 100,
		}
		result.AddConditionResult(condResult1)

		condResult2 := ConditionResult{
			Passed:      true,
			Description: "Condition 2",
			Type:        MesoCondition,
			Operator:    GreaterThan,
			Value:       1000,
			ActualValue: 2000,
		}
		result.AddConditionResult(condResult2)

		condResult3 := ConditionResult{
			Passed:      false,
			Description: "Condition 3",
			Type:        FameCondition,
			Operator:    GreaterEqual,
			Value:       50,
			ActualValue: 40,
		}
		result.AddConditionResult(condResult3)

		if result.Passed() {
			t.Errorf("After mixed AddConditionResult calls passed = %v, want false", result.Passed())
		}

		if len(result.Details()) != 3 {
			t.Errorf("After multiple AddConditionResult calls details length = %v, want 3", len(result.Details()))
		}
	})
}
