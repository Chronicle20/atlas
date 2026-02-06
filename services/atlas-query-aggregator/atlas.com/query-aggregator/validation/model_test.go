package validation

import (
	"atlas-query-aggregator/asset"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/compartment"
	"atlas-query-aggregator/guild"
	"atlas-query-aggregator/guild/member"
	"atlas-query-aggregator/guild/title"
	"atlas-query-aggregator/inventory"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/quest"
	"strings"
	"testing"
	"time"

	inventory_type "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   2000001,
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
				referenceId:   9999999, // Non-existent item
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
		// Guild condition tests - character not in guild
		{
			name: "Guild ID - character not in guild",
			condition: Condition{
				conditionType: GuildIdCondition,
				operator:      Equals,
				value:         1001,
			},
			wantPassed:   false,
			wantContains: "Guild ID = 1001 (character not in guild)",
		},
		{
			name: "Guild Rank - character not in guild",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      Equals,
				value:         1,
			},
			wantPassed:   false,
			wantContains: "Guild Rank = 1 (character not in guild)",
		},
		// Test new condition types that require context
		{
			name: "Quest Status - requires context",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         1,
				referenceId:   1001,
			},
			wantPassed:   false,
			wantContains: "Quest 1001 Status validation requires ValidationContext",
		},
		{
			name: "Quest Progress - requires context",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         1,
				referenceId:   1001,
				step:          "step1",
			},
			wantPassed:   false,
			wantContains: "Quest 1001 Progress validation (step: step1) requires ValidationContext",
		},
		{
			name: "Marriage Gifts - requires context",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         1,
			},
			wantPassed:   false,
			wantContains: "Unclaimed Marriage Gifts validation requires ValidationContext",
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

func TestCondition_EvaluateWithContext(t *testing.T) {
	// Create test inventory with items
	compartmentId := uuid.New()
	consumableCompartment := createTestCompartment(compartmentId, 123, inventory_type.TypeValueUse, 100)

	// Create inventory model
	inventoryModel := inventory.NewBuilder(123).
		SetConsumable(consumableCompartment).
		Build()

	// Create a test character
	character := character.NewModelBuilder().
		SetId(123).
		SetJobId(100).
		SetMeso(10000).
		SetMapId(2000).
		SetFame(50).
		SetGender(0).
		SetLevel(25).
		SetReborns(3).
		SetDojoPoints(1500).
		SetVanquisherKills(7).
		SetInventory(inventoryModel).
		Build()

	// Create test quest models
	questStarted := quest.NewModelBuilder().
		SetId(1001).
		SetStatus(quest.STARTED).
		SetProgress("step1", 5).
		SetProgress("step2", 10).
		Build()

	questCompleted := quest.NewModelBuilder().
		SetId(1002).
		SetStatus(quest.COMPLETED).
		SetProgress("final", 100).
		Build()

	// Create test marriage model with unclaimed gifts
	marriageWithGifts := marriage.NewModelBuilder().
		SetCharacterId(123).
		SetHasUnclaimedGifts(true).
		SetUnclaimedGiftCount(3).
		Build()

	// Create test marriage model without gifts
	marriageNoGifts := marriage.NewModelBuilder().
		SetCharacterId(123).
		SetHasUnclaimedGifts(false).
		SetUnclaimedGiftCount(0).
		Build()

	// Create validation context with quest and marriage data
	contextWithData := NewValidationContextBuilder(character).
		AddQuest(questStarted).
		AddQuest(questCompleted).
		SetMarriage(marriageWithGifts).
		Build()

	// Create validation context without marriage gifts
	contextNoGifts := NewValidationContextBuilder(character).
		AddQuest(questStarted).
		AddQuest(questCompleted).
		SetMarriage(marriageNoGifts).
		Build()

	tests := []struct {
		name         string
		condition    Condition
		context      ValidationContext
		wantPassed   bool
		wantContains string
	}{
		// Quest Status condition tests
		{
			name: "Quest Status STARTED - pass",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.STARTED),
				referenceId:   1001,
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Quest 1001 Status = 1",
		},
		{
			name: "Quest Status STARTED - fail",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.COMPLETED),
				referenceId:   1001,
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Quest 1001 Status = 2",
		},
		{
			name: "Quest Status COMPLETED - pass",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.COMPLETED),
				referenceId:   1002,
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Quest 1002 Status = 2",
		},
		{
			name: "Quest Status - quest not found",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.STARTED),
				referenceId:   9999,
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Quest 9999 not found",
		},
		// Quest Progress condition tests
		{
			name: "Quest Progress step1 - pass",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         5,
				referenceId:   1001,
				step:          "step1",
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Quest 1001 Progress (step: step1) = 5",
		},
		{
			name: "Quest Progress step1 - fail",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         10,
				referenceId:   1001,
				step:          "step1",
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Quest 1001 Progress (step: step1) = 10",
		},
		{
			name: "Quest Progress step2 greater than - pass",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      GreaterThan,
				value:         5,
				referenceId:   1001,
				step:          "step2",
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Quest 1001 Progress (step: step2) > 5",
		},
		{
			name: "Quest Progress step2 greater than - fail",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      GreaterThan,
				value:         15,
				referenceId:   1001,
				step:          "step2",
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Quest 1001 Progress (step: step2) > 15",
		},
		{
			name: "Quest Progress nonexistent step - returns 0",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         0,
				referenceId:   1001,
				step:          "nonexistent",
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Quest 1001 Progress (step: nonexistent) = 0",
		},
		{
			name: "Quest Progress - quest not found",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         5,
				referenceId:   9999,
				step:          "step1",
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Quest 9999 not found",
		},
		// Marriage gifts condition tests
		{
			name: "Marriage Gifts - has gifts (1)",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         1,
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Unclaimed Marriage Gifts = 1",
		},
		{
			name: "Marriage Gifts - has gifts (0)",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         0,
			},
			context:      contextWithData,
			wantPassed:   false,
			wantContains: "Unclaimed Marriage Gifts = 0",
		},
		{
			name: "Marriage Gifts - no gifts (0)",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         0,
			},
			context:      contextNoGifts,
			wantPassed:   true,
			wantContains: "Unclaimed Marriage Gifts = 0",
		},
		{
			name: "Marriage Gifts - no gifts (1)",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         1,
			},
			context:      contextNoGifts,
			wantPassed:   false,
			wantContains: "Unclaimed Marriage Gifts = 1",
		},
		// Test that existing conditions still work with context
		{
			name: "Level condition with context - pass",
			condition: Condition{
				conditionType: LevelCondition,
				operator:      Equals,
				value:         25,
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Level = 25",
		},
		{
			name: "Item condition with context - pass",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         10,
				referenceId:   2000001,
			},
			context:      contextWithData,
			wantPassed:   true,
			wantContains: "Item 2000001 quantity = 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.EvaluateWithContext(tt.context)

			if result.Passed != tt.wantPassed {
				t.Errorf("Condition.EvaluateWithContext() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Description != tt.wantContains {
				t.Errorf("Condition.EvaluateWithContext() description = %v, want %v", result.Description, tt.wantContains)
			}
		})
	}
}

func TestValidationContext(t *testing.T) {
	// Create test character
	character := character.NewModelBuilder().
		SetId(123).
		SetLevel(25).
		Build()

	// Create test quest models
	quest1 := quest.NewModelBuilder().
		SetId(1001).
		SetStatus(quest.STARTED).
		SetProgress("step1", 5).
		Build()

	quest2 := quest.NewModelBuilder().
		SetId(1002).
		SetStatus(quest.COMPLETED).
		Build()

	// Create test marriage model
	marriage := marriage.NewModelBuilder().
		SetCharacterId(123).
		SetHasUnclaimedGifts(true).
		Build()

	t.Run("NewValidationContext", func(t *testing.T) {
		ctx := NewValidationContext(character)

		if ctx.Character().Id() != 123 {
			t.Errorf("NewValidationContext() character ID = %v, want 123", ctx.Character().Id())
		}

		if ctx.Marriage().CharacterId() != 123 {
			t.Errorf("NewValidationContext() marriage character ID = %v, want 123", ctx.Marriage().CharacterId())
		}

		if ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("NewValidationContext() marriage has gifts = %v, want false", ctx.Marriage().HasUnclaimedGifts())
		}

		// Test that quest doesn't exist
		_, exists := ctx.Quest(1001)
		if exists {
			t.Errorf("NewValidationContext() quest 1001 exists = %v, want false", exists)
		}
	})

	t.Run("WithQuest", func(t *testing.T) {
		ctx := NewValidationContext(character)
		ctx = ctx.WithQuest(quest1)

		questModel, exists := ctx.Quest(1001)
		if !exists {
			t.Errorf("WithQuest() quest 1001 exists = %v, want true", exists)
		}

		if questModel.Id() != 1001 {
			t.Errorf("WithQuest() quest ID = %v, want 1001", questModel.Id())
		}

		if questModel.Status() != quest.STARTED {
			t.Errorf("WithQuest() quest status = %v, want STARTED", questModel.Status())
		}

		// Test that other quest doesn't exist
		_, exists = ctx.Quest(1002)
		if exists {
			t.Errorf("WithQuest() quest 1002 exists = %v, want false", exists)
		}
	})

	t.Run("WithMarriage", func(t *testing.T) {
		ctx := NewValidationContext(character)
		ctx = ctx.WithMarriage(marriage)

		if ctx.Marriage().CharacterId() != 123 {
			t.Errorf("WithMarriage() marriage character ID = %v, want 123", ctx.Marriage().CharacterId())
		}

		if !ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("WithMarriage() marriage has gifts = %v, want true", ctx.Marriage().HasUnclaimedGifts())
		}
	})

	t.Run("Multiple quests", func(t *testing.T) {
		ctx := NewValidationContext(character)
		ctx = ctx.WithQuest(quest1)
		ctx = ctx.WithQuest(quest2)

		// Test both quests exist
		questModel1, exists1 := ctx.Quest(1001)
		if !exists1 {
			t.Errorf("Multiple quests - quest 1001 exists = %v, want true", exists1)
		}

		questModel2, exists2 := ctx.Quest(1002)
		if !exists2 {
			t.Errorf("Multiple quests - quest 1002 exists = %v, want true", exists2)
		}

		if questModel1.Status() != quest.STARTED {
			t.Errorf("Multiple quests - quest 1001 status = %v, want STARTED", questModel1.Status())
		}

		if questModel2.Status() != quest.COMPLETED {
			t.Errorf("Multiple quests - quest 1002 status = %v, want COMPLETED", questModel2.Status())
		}
	})
}

func TestValidationContextBuilder(t *testing.T) {
	// Create test character
	character := character.NewModelBuilder().
		SetId(123).
		SetLevel(25).
		Build()

	// Create test quest models
	quest1 := quest.NewModelBuilder().
		SetId(1001).
		SetStatus(quest.STARTED).
		SetProgress("step1", 5).
		Build()

	quest2 := quest.NewModelBuilder().
		SetId(1002).
		SetStatus(quest.COMPLETED).
		Build()

	// Create test marriage model
	marriage := marriage.NewModelBuilder().
		SetCharacterId(123).
		SetHasUnclaimedGifts(true).
		SetUnclaimedGiftCount(2).
		Build()

	t.Run("NewValidationContextBuilder", func(t *testing.T) {
		builder := NewValidationContextBuilder(character)
		ctx := builder.Build()

		if ctx.Character().Id() != 123 {
			t.Errorf("NewValidationContextBuilder() character ID = %v, want 123", ctx.Character().Id())
		}

		if ctx.Marriage().CharacterId() != 123 {
			t.Errorf("NewValidationContextBuilder() marriage character ID = %v, want 123", ctx.Marriage().CharacterId())
		}

		if ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("NewValidationContextBuilder() marriage has gifts = %v, want false", ctx.Marriage().HasUnclaimedGifts())
		}
	})

	t.Run("AddQuest", func(t *testing.T) {
		builder := NewValidationContextBuilder(character)
		builder.AddQuest(quest1)
		ctx := builder.Build()

		questModel, exists := ctx.Quest(1001)
		if !exists {
			t.Errorf("AddQuest() quest 1001 exists = %v, want true", exists)
		}

		if questModel.Status() != quest.STARTED {
			t.Errorf("AddQuest() quest status = %v, want STARTED", questModel.Status())
		}
	})

	t.Run("SetMarriage", func(t *testing.T) {
		builder := NewValidationContextBuilder(character)
		builder.SetMarriage(marriage)
		ctx := builder.Build()

		if !ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("SetMarriage() marriage has gifts = %v, want true", ctx.Marriage().HasUnclaimedGifts())
		}

		if ctx.Marriage().UnclaimedGiftCount() != 2 {
			t.Errorf("SetMarriage() marriage gift count = %v, want 2", ctx.Marriage().UnclaimedGiftCount())
		}
	})

	t.Run("Builder fluent interface", func(t *testing.T) {
		ctx := NewValidationContextBuilder(character).
			AddQuest(quest1).
			AddQuest(quest2).
			SetMarriage(marriage).
			Build()

		// Test character
		if ctx.Character().Id() != 123 {
			t.Errorf("Builder fluent interface - character ID = %v, want 123", ctx.Character().Id())
		}

		// Test quests
		questModel1, exists1 := ctx.Quest(1001)
		if !exists1 {
			t.Errorf("Builder fluent interface - quest 1001 exists = %v, want true", exists1)
		}

		questModel2, exists2 := ctx.Quest(1002)
		if !exists2 {
			t.Errorf("Builder fluent interface - quest 1002 exists = %v, want true", exists2)
		}

		if questModel1.Status() != quest.STARTED {
			t.Errorf("Builder fluent interface - quest 1001 status = %v, want STARTED", questModel1.Status())
		}

		if questModel2.Status() != quest.COMPLETED {
			t.Errorf("Builder fluent interface - quest 1002 status = %v, want COMPLETED", questModel2.Status())
		}

		// Test marriage
		if !ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("Builder fluent interface - marriage has gifts = %v, want true", ctx.Marriage().HasUnclaimedGifts())
		}

		if ctx.Marriage().UnclaimedGiftCount() != 2 {
			t.Errorf("Builder fluent interface - marriage gift count = %v, want 2", ctx.Marriage().UnclaimedGiftCount())
		}
	})
}

func TestCondition_Evaluate_WithGuild(t *testing.T) {
	// Create test inventory with items
	compartmentId := uuid.New()
	consumableCompartment := createTestCompartment(compartmentId, 123, inventory_type.TypeValueUse, 100)

	// Create inventory model
	inventoryModel := inventory.NewBuilder(123).
		SetConsumable(consumableCompartment).
		Build()

	// Create a guild for the character
	guildRestModel := guild.RestModel{
		Id:                  1001,
		WorldId:             1,
		Name:                "TestGuild",
		Notice:              "Test guild notice",
		Points:              1000,
		Capacity:            100,
		Logo:                1,
		LogoColor:           1,
		LogoBackground:      1,
		LogoBackgroundColor: 1,
		LeaderId:            123,
		Members: []member.RestModel{
			{
				CharacterId:   123,
				Name:          "TestMember",
				JobId:         100,
				Level:         25,
				Title:         3,
				Online:        true,
				AllianceTitle: 0,
			},
		},
		Titles: []title.RestModel{},
	}
	guildModel, _ := guild.Extract(guildRestModel)

	// Create a test character with guild
	character := character.NewModelBuilder().
		SetId(123).
		SetInventory(inventoryModel).
		SetGuild(guildModel).
		Build()

	tests := []struct {
		name         string
		condition    Condition
		wantPassed   bool
		wantContains string
	}{
		{
			name: "Guild ID - pass",
			condition: Condition{
				conditionType: GuildIdCondition,
				operator:      Equals,
				value:         1001,
			},
			wantPassed:   true,
			wantContains: "Guild ID = 1001",
		},
		{
			name: "Guild ID - fail",
			condition: Condition{
				conditionType: GuildIdCondition,
				operator:      Equals,
				value:         2001,
			},
			wantPassed:   false,
			wantContains: "Guild ID = 2001",
		},
		{
			name: "Guild Rank - pass",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      Equals,
				value:         3,
			},
			wantPassed:   true,
			wantContains: "Guild Rank = 3",
		},
		{
			name: "Guild Rank - fail",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      Equals,
				value:         1,
			},
			wantPassed:   false,
			wantContains: "Guild Rank = 1",
		},
		{
			name: "Guild Rank greater than - pass",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      GreaterThan,
				value:         2,
			},
			wantPassed:   true,
			wantContains: "Guild Rank > 2",
		},
		{
			name: "Guild Rank greater than - fail",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      GreaterThan,
				value:         5,
			},
			wantPassed:   false,
			wantContains: "Guild Rank > 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.Evaluate(character)

			if result.Passed != tt.wantPassed {
				t.Errorf("Condition.Evaluate() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Description != tt.wantContains {
				t.Errorf("Condition.Evaluate() description = %v, want %v", result.Description, tt.wantContains)
			}
		})
	}
}

// TestCondition_ErrorHandling tests error scenarios for missing/invalid data
func TestCondition_ErrorHandling(t *testing.T) {
	// Create minimal test character for error cases
	character := character.NewModelBuilder().
		SetId(123).
		SetLevel(25).
		SetJobId(100).
		Build()

	tests := []struct {
		name         string
		condition    Condition
		wantPassed   bool
		wantContains string
		wantError    bool
	}{
		// Test unsupported condition type
		{
			name: "Unsupported condition type",
			condition: Condition{
				conditionType: ConditionType("unsupported"),
				operator:      Equals,
				value:         100,
			},
			wantPassed:   false,
			wantContains: "Unsupported condition type: unsupported",
			wantError:    true,
		},
		// Test invalid item ID
		{
			name: "Invalid item ID",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         10,
				referenceId:   99999999, // Invalid item ID
			},
			wantPassed:   false,
			wantContains: "Invalid item ID: 99999999",
			wantError:    true,
		},
		// Test character not in guild for guild ID condition
		{
			name: "Character not in guild - Guild ID condition",
			condition: Condition{
				conditionType: GuildIdCondition,
				operator:      Equals,
				value:         1001,
			},
			wantPassed:   false,
			wantContains: "Guild ID = 1001 (character not in guild)",
			wantError:    true,
		},
		// Test character not in guild for guild rank condition
		{
			name: "Character not in guild - Guild Rank condition",
			condition: Condition{
				conditionType: GuildRankCondition,
				operator:      Equals,
				value:         3,
			},
			wantPassed:   false,
			wantContains: "Guild Rank = 3 (character not in guild)",
			wantError:    true,
		},
		// Test quest conditions without context
		{
			name: "Quest Status without context",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.STARTED),
				referenceId:   1001,
			},
			wantPassed:   false,
			wantContains: "Quest 1001 Status validation requires ValidationContext",
			wantError:    true,
		},
		{
			name: "Quest Progress without context",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         5,
				referenceId:   1001,
				step:          "step1",
			},
			wantPassed:   false,
			wantContains: "Quest 1001 Progress validation (step: step1) requires ValidationContext",
			wantError:    true,
		},
		// Test marriage condition without context
		{
			name: "Marriage Gifts without context",
			condition: Condition{
				conditionType: UnclaimedMarriageGiftsCondition,
				operator:      Equals,
				value:         1,
			},
			wantPassed:   false,
			wantContains: "Unclaimed Marriage Gifts validation requires ValidationContext",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.Evaluate(character)

			if result.Passed != tt.wantPassed {
				t.Errorf("Condition.Evaluate() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Description != tt.wantContains {
				t.Errorf("Condition.Evaluate() description = %v, want %v", result.Description, tt.wantContains)
			}

			// For error cases, verify that the condition correctly identified the error
			if tt.wantError && result.Passed {
				t.Errorf("Expected error condition to fail, but it passed")
			}
		})
	}
}

// TestConditionBuilder_ErrorHandling tests error scenarios for condition builder
func TestConditionBuilder_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         ConditionInput
		wantError     bool
		errorContains string
	}{
		// Test invalid condition type
		{
			name: "Invalid condition type",
			input: ConditionInput{
				Type:     "invalidType",
				Operator: "=",
				Value:    100,
			},
			wantError:     true,
			errorContains: "unsupported condition type",
		},
		// Test invalid operator
		{
			name: "Invalid operator",
			input: ConditionInput{
				Type:     "level",
				Operator: "!=",
				Value:    25,
			},
			wantError:     true,
			errorContains: "unsupported operator",
		},
		// Test item condition without referenceId
		{
			name: "Item condition without referenceId",
			input: ConditionInput{
				Type:     "item",
				Operator: "=",
				Value:    10,
				// Missing ReferenceId
			},
			wantError:     true,
			errorContains: "referenceId is required for item conditions",
		},
		// Test quest status condition without referenceId
		{
			name: "Quest status condition without referenceId",
			input: ConditionInput{
				Type:     "questStatus",
				Operator: "=",
				Value:    int(quest.STARTED),
				// Missing ReferenceId
			},
			wantError:     true,
			errorContains: "referenceId is required for quest conditions",
		},
		// Test quest progress condition without referenceId
		{
			name: "Quest progress condition without referenceId",
			input: ConditionInput{
				Type:     "questProgress",
				Operator: "=",
				Value:    5,
				Step:     "step1",
				// Missing ReferenceId
			},
			wantError:     true,
			errorContains: "referenceId is required for quest conditions",
		},
		// Test quest progress condition without step
		{
			name: "Quest progress condition without step",
			input: ConditionInput{
				Type:        "questProgress",
				Operator:    "=",
				Value:       5,
				ReferenceId: 1001,
				// Missing Step
			},
			wantError:     true,
			errorContains: "step is required for quest progress conditions",
		},
		// Test empty condition type
		{
			name: "Empty condition type",
			input: ConditionInput{
				Type:     "",
				Operator: "=",
				Value:    100,
			},
			wantError:     true,
			errorContains: "unsupported condition type",
		},
		// Test empty operator
		{
			name: "Empty operator",
			input: ConditionInput{
				Type:     "level",
				Operator: "",
				Value:    25,
			},
			wantError:     true,
			errorContains: "unsupported operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewConditionBuilder()
			_, err := builder.FromInput(tt.input).Build()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestConditionWithContext_ErrorHandling tests error scenarios with validation context
func TestConditionWithContext_ErrorHandling(t *testing.T) {
	// Create minimal test character
	character := character.NewModelBuilder().
		SetId(123).
		SetLevel(25).
		Build()

	// Create empty validation context (no quests, no marriage data)
	emptyContext := NewValidationContext(character)

	tests := []struct {
		name         string
		condition    Condition
		context      ValidationContext
		wantPassed   bool
		wantContains string
		wantError    bool
	}{
		// Test quest not found
		{
			name: "Quest Status - quest not found",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.STARTED),
				referenceId:   9999,
			},
			context:      emptyContext,
			wantPassed:   false,
			wantContains: "Quest 9999 not found",
			wantError:    true,
		},
		{
			name: "Quest Progress - quest not found",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         5,
				referenceId:   9999,
				step:          "step1",
			},
			context:      emptyContext,
			wantPassed:   false,
			wantContains: "Quest 9999 not found",
			wantError:    true,
		},
		// Test zero reference ID scenarios
		{
			name: "Quest Status - zero reference ID",
			condition: Condition{
				conditionType: QuestStatusCondition,
				operator:      Equals,
				value:         int(quest.STARTED),
				referenceId:   0,
			},
			context:      emptyContext,
			wantPassed:   false,
			wantContains: "Quest 0 not found",
			wantError:    true,
		},
		{
			name: "Item condition - zero reference ID",
			condition: Condition{
				conditionType: ItemCondition,
				operator:      Equals,
				value:         10,
				referenceId:   0,
			},
			context:      emptyContext,
			wantPassed:   false,
			wantContains: "Invalid item ID: 0",
			wantError:    true,
		},
		// Test boundary conditions
		{
			name: "Quest Progress - empty step",
			condition: Condition{
				conditionType: QuestProgressCondition,
				operator:      Equals,
				value:         5,
				referenceId:   1001,
				step:          "", // Empty step
			},
			context:      emptyContext,
			wantPassed:   false,
			wantContains: "Quest 1001 not found",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.EvaluateWithContext(tt.context)

			if result.Passed != tt.wantPassed {
				t.Errorf("Condition.EvaluateWithContext() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Description != tt.wantContains {
				t.Errorf("Condition.EvaluateWithContext() description = %v, want %v", result.Description, tt.wantContains)
			}

			// For error cases, verify that the condition correctly identified the error
			if tt.wantError && result.Passed {
				t.Errorf("Expected error condition to fail, but it passed")
			}
		})
	}
}

// TestValidationContext_ErrorHandling tests error scenarios with validation context creation
func TestValidationContext_ErrorHandling(t *testing.T) {
	// Create minimal test character
	character := character.NewModelBuilder().
		SetId(123).
		SetLevel(25).
		Build()

	t.Run("Empty validation context", func(t *testing.T) {
		ctx := NewValidationContext(character)

		// Test quest retrieval from empty context
		_, exists := ctx.Quest(1001)
		if exists {
			t.Errorf("Expected quest 1001 to not exist in empty context")
		}

		// Test that marriage data is properly initialized
		if ctx.Marriage().CharacterId() != 123 {
			t.Errorf("Expected marriage character ID to be 123, got %d", ctx.Marriage().CharacterId())
		}

		if ctx.Marriage().HasUnclaimedGifts() {
			t.Errorf("Expected empty marriage context to have no unclaimed gifts")
		}
	})

	t.Run("Validation context with nil quest", func(t *testing.T) {
		ctx := NewValidationContext(character)

		// Test that requesting a non-existent quest returns appropriate values
		for i := uint32(0); i < 10; i++ {
			_, exists := ctx.Quest(i)
			if exists {
				t.Errorf("Expected quest %d to not exist in empty context", i)
			}
		}
	})
}
