package factory

import (
	"atlas-character-factory/configuration/tenant/characters/template"
	"atlas-character-factory/saga"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildCharacterCreationSaga(t *testing.T) {
	tests := []struct {
		name     string
		input    RestModel
		template template.RestModel
		validate func(t *testing.T, result saga.Saga)
	}{
		{
			name: "basic character creation saga",
			input: RestModel{
				AccountId:   1001,
				WorldId:     0,
				Name:        "TestCharacter",
				Gender:      0,
				JobIndex:    100,
				SubJobIndex: 0,
				Face:        20000,
				Hair:        30000,
				HairColor:   7,
				SkinColor:   0,
				Top:         1040002,
				Bottom:      1060002,
				Shoes:       1072001,
				Weapon:      1302000,
			},
			template: template.RestModel{
				JobIndex:    100,
				SubJobIndex: 0,
				MapId:       10000,
				Gender:      0,
				Items:       []uint32{2000000, 2000001, 2000002}, // Sample items
				Skills:      []uint32{1000, 1001},                // Sample skills
			},
			validate: func(t *testing.T, result saga.Saga) {
				// Verify saga metadata
				if result.SagaType != saga.CharacterCreation {
					t.Errorf("Expected saga type %v, got %v", saga.CharacterCreation, result.SagaType)
				}

				if result.InitiatedBy != "account_1001" {
					t.Errorf("Expected initiatedBy 'account_1001', got '%s'", result.InitiatedBy)
				}

				// Verify we have the expected number of steps
				// 1 create character + 3 award items + 4 equip assets + 2 create skills = 10 steps
				expectedSteps := 1 + 3 + 4 + 2
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				// Verify first step is character creation
				firstStep := result.Steps[0]
				if firstStep.StepId != "create" {
					t.Errorf("Expected first step ID 'create', got '%s'", firstStep.StepId)
				}
				if firstStep.Action != saga.CreateCharacter {
					t.Errorf("Expected first step action %v, got %v", saga.CreateCharacter, firstStep.Action)
				}
				if firstStep.Status != saga.Pending {
					t.Errorf("Expected first step status %v, got %v", saga.Pending, firstStep.Status)
				}

				// Verify character creation payload
				if payload, ok := firstStep.Payload.(saga.CharacterCreatePayload); ok {
					if payload.AccountId != 1001 {
						t.Errorf("Expected AccountId 1001, got %d", payload.AccountId)
					}
					if payload.Name != "TestCharacter" {
						t.Errorf("Expected Name 'TestCharacter', got '%s'", payload.Name)
					}
					if payload.WorldId != 0 {
						t.Errorf("Expected WorldId 0, got %d", payload.WorldId)
					}
					if payload.JobId != 100 {
						t.Errorf("Expected JobId 100, got %d", payload.JobId)
					}
					if payload.Face != 20000 {
						t.Errorf("Expected Face 20000, got %d", payload.Face)
					}
					if payload.Hair != 30000 {
						t.Errorf("Expected Hair 30000, got %d", payload.Hair)
					}
					if payload.HairColor != 7 {
						t.Errorf("Expected HairColor 7, got %d", payload.HairColor)
					}
					if payload.Skin != 0 {
						t.Errorf("Expected Skin 0, got %d", payload.Skin)
					}
					if payload.Top != 1040002 {
						t.Errorf("Expected Top 1040002, got %d", payload.Top)
					}
					if payload.Bottom != 1060002 {
						t.Errorf("Expected Bottom 1060002, got %d", payload.Bottom)
					}
					if payload.Shoes != 1072001 {
						t.Errorf("Expected Shoes 1072001, got %d", payload.Shoes)
					}
					if payload.Weapon != 1302000 {
						t.Errorf("Expected Weapon 1302000, got %d", payload.Weapon)
					}
				} else {
					t.Error("First step payload is not CharacterCreatePayload")
				}

				// Verify award item steps
				for i := 0; i < 3; i++ {
					stepIndex := 1 + i
					step := result.Steps[stepIndex]
					expectedStepId := "award_item_" + string(rune('0'+i))
					if step.StepId != expectedStepId {
						t.Errorf("Expected step ID '%s', got '%s'", expectedStepId, step.StepId)
					}
					if step.Action != saga.AwardAsset {
						t.Errorf("Expected step action %v, got %v", saga.AwardAsset, step.Action)
					}
					if step.Status != saga.Pending {
						t.Errorf("Expected step status %v, got %v", saga.Pending, step.Status)
					}

					if payload, ok := step.Payload.(saga.AwardItemActionPayload); ok {
						if payload.CharacterId != 0 {
							t.Errorf("Expected CharacterId 0 (to be set by orchestrator), got %d", payload.CharacterId)
						}
						expectedTemplateId := uint32(2000000 + i)
						if payload.Item.TemplateId != expectedTemplateId {
							t.Errorf("Expected TemplateId %d, got %d", expectedTemplateId, payload.Item.TemplateId)
						}
						if payload.Item.Quantity != 1 {
							t.Errorf("Expected Quantity 1, got %d", payload.Item.Quantity)
						}
					} else {
						t.Errorf("Step %d payload is not AwardItemActionPayload", stepIndex)
					}
				}

				// Verify equipment steps
				equipmentNames := []string{"top", "bottom", "shoes", "weapon"}
				equipmentTemplateIds := []uint32{1040002, 1060002, 1072001, 1302000}
				for i := 0; i < 4; i++ {
					stepIndex := 4 + i
					step := result.Steps[stepIndex]
					expectedStepId := "equip_" + equipmentNames[i]
					if step.StepId != expectedStepId {
						t.Errorf("Expected step ID '%s', got '%s'", expectedStepId, step.StepId)
					}
					if step.Action != saga.CreateAndEquipAsset {
						t.Errorf("Expected step action %v, got %v", saga.CreateAndEquipAsset, step.Action)
					}
					if step.Status != saga.Pending {
						t.Errorf("Expected step status %v, got %v", saga.Pending, step.Status)
					}

					if payload, ok := step.Payload.(saga.CreateAndEquipAssetPayload); ok {
						if payload.CharacterId != 0 {
							t.Errorf("Expected CharacterId 0 (to be set by orchestrator), got %d", payload.CharacterId)
						}
						if payload.TemplateId != equipmentTemplateIds[i] {
							t.Errorf("Expected TemplateId %d, got %d", equipmentTemplateIds[i], payload.TemplateId)
						}
						if payload.Source != 1 {
							t.Errorf("Expected Source 1, got %d", payload.Source)
						}
						if payload.Destination != 0 {
							t.Errorf("Expected Destination 0, got %d", payload.Destination)
						}
					} else {
						t.Errorf("Step %d payload is not CreateAndEquipAssetPayload", stepIndex)
					}
				}

				// Verify skill creation steps
				for i := 0; i < 2; i++ {
					stepIndex := 8 + i
					step := result.Steps[stepIndex]
					expectedStepId := "create_skill_" + string(rune('0'+i))
					if step.StepId != expectedStepId {
						t.Errorf("Expected step ID '%s', got '%s'", expectedStepId, step.StepId)
					}
					if step.Action != saga.CreateSkill {
						t.Errorf("Expected step action %v, got %v", saga.CreateSkill, step.Action)
					}
					if step.Status != saga.Pending {
						t.Errorf("Expected step status %v, got %v", saga.Pending, step.Status)
					}

					if payload, ok := step.Payload.(saga.CreateSkillPayload); ok {
						if payload.CharacterId != 0 {
							t.Errorf("Expected CharacterId 0 (to be set by orchestrator), got %d", payload.CharacterId)
						}
						expectedSkillId := uint32(1000 + i)
						if payload.SkillId != expectedSkillId {
							t.Errorf("Expected SkillId %d, got %d", expectedSkillId, payload.SkillId)
						}
						if payload.Level != 1 {
							t.Errorf("Expected Level 1, got %d", payload.Level)
						}
						if payload.MasterLevel != 0 {
							t.Errorf("Expected MasterLevel 0, got %d", payload.MasterLevel)
						}
						if !payload.Expiration.IsZero() {
							t.Errorf("Expected zero Expiration, got %v", payload.Expiration)
						}
					} else {
						t.Errorf("Step %d payload is not CreateSkillPayload", stepIndex)
					}
				}
			},
		},
		{
			name: "character creation with zero equipment",
			input: RestModel{
				AccountId:   2001,
				WorldId:     1,
				Name:        "MinimalChar",
				Gender:      1,
				JobIndex:    200,
				SubJobIndex: 1,
				Face:        21000,
				Hair:        31000,
				HairColor:   5,
				SkinColor:   2,
				Top:         0, // No equipment
				Bottom:      0,
				Shoes:       0,
				Weapon:      0,
			},
			template: template.RestModel{
				JobIndex:    200,
				SubJobIndex: 1,
				MapId:       20000,
				Gender:      1,
				Items:       []uint32{}, // No items
				Skills:      []uint32{}, // No skills
			},
			validate: func(t *testing.T, result saga.Saga) {
				// Should only have the character creation step (no award, equip, or skill steps)
				expectedSteps := 1
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				// Verify it's just the character creation step
				firstStep := result.Steps[0]
				if firstStep.StepId != "create" {
					t.Errorf("Expected first step ID 'create', got '%s'", firstStep.StepId)
				}
				if firstStep.Action != saga.CreateCharacter {
					t.Errorf("Expected first step action %v, got %v", saga.CreateCharacter, firstStep.Action)
				}
			},
		},
		{
			name: "character creation with partial equipment",
			input: RestModel{
				AccountId:   3001,
				WorldId:     2,
				Name:        "PartialEquipChar",
				Gender:      0,
				JobIndex:    300,
				SubJobIndex: 0,
				Face:        22000,
				Hair:        32000,
				HairColor:   3,
				SkinColor:   1,
				Top:         1040003, // Only top and weapon
				Bottom:      0,
				Shoes:       0,
				Weapon:      1302001,
			},
			template: template.RestModel{
				JobIndex:    300,
				SubJobIndex: 0,
				MapId:       30000,
				Gender:      0,
				Items:       []uint32{2000003},
				Skills:      []uint32{1002},
			},
			validate: func(t *testing.T, result saga.Saga) {
				// 1 create character + 1 award item + 2 equip assets (top, weapon) + 1 create skill = 5 steps
				expectedSteps := 1 + 1 + 2 + 1
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				// Check that we only have equip steps for top and weapon
				equipStepFound := map[string]bool{"equip_top": false, "equip_weapon": false}
				for _, step := range result.Steps[2:4] { // Skip create and award steps
					if step.Action == saga.CreateAndEquipAsset {
						equipStepFound[step.StepId] = true
					}
				}

				if !equipStepFound["equip_top"] {
					t.Error("Expected equip_top step not found")
				}
				if !equipStepFound["equip_weapon"] {
					t.Error("Expected equip_weapon step not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactionId := uuid.New()
			result := buildCharacterCreationSaga(transactionId, tt.input, tt.template)

			// Verify transaction ID
			if result.TransactionId != transactionId {
				t.Errorf("Expected TransactionId %v, got %v", transactionId, result.TransactionId)
			}

			// Run custom validation
			tt.validate(t, result)
		})
	}
}

func TestBuildCharacterCreationSaga_StepOrdering(t *testing.T) {
	input := RestModel{
		AccountId:   1001,
		WorldId:     0,
		Name:        "OrderTestChar",
		Gender:      0,
		JobIndex:    100,
		SubJobIndex: 0,
		Face:        20000,
		Hair:        30000,
		HairColor:   7,
		SkinColor:   0,
		Top:         1040002,
		Bottom:      1060002,
		Shoes:       1072001,
		Weapon:      1302000,
	}

	template := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000, 2000001},
		Skills:      []uint32{1000, 1001},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, template)

	// Verify step ordering: create → award items → equip assets → create skills
	expectedStepOrder := []struct {
		stepType string
		action   saga.Action
	}{
		{"create", saga.CreateCharacter},
		{"award_item_0", saga.AwardAsset},
		{"award_item_1", saga.AwardAsset},
		{"equip_top", saga.CreateAndEquipAsset},
		{"equip_bottom", saga.CreateAndEquipAsset},
		{"equip_shoes", saga.CreateAndEquipAsset},
		{"equip_weapon", saga.CreateAndEquipAsset},
		{"create_skill_0", saga.CreateSkill},
		{"create_skill_1", saga.CreateSkill},
	}

	if len(result.Steps) != len(expectedStepOrder) {
		t.Fatalf("Expected %d steps, got %d", len(expectedStepOrder), len(result.Steps))
	}

	for i, expected := range expectedStepOrder {
		step := result.Steps[i]
		if step.StepId != expected.stepType {
			t.Errorf("Step %d: expected StepId '%s', got '%s'", i, expected.stepType, step.StepId)
		}
		if step.Action != expected.action {
			t.Errorf("Step %d: expected Action %v, got %v", i, expected.action, step.Action)
		}
		if step.Status != saga.Pending {
			t.Errorf("Step %d: expected Status %v, got %v", i, saga.Pending, step.Status)
		}
		if step.CreatedAt.IsZero() {
			t.Errorf("Step %d: CreatedAt should not be zero", i)
		}
		if step.UpdatedAt.IsZero() {
			t.Errorf("Step %d: UpdatedAt should not be zero", i)
		}
	}
}

func TestBuildCharacterCreationSaga_EmptyTemplate(t *testing.T) {
	input := RestModel{
		AccountId:   4001,
		WorldId:     0,
		Name:        "EmptyTemplateChar",
		Gender:      0,
		JobIndex:    400,
		SubJobIndex: 0,
		Face:        20000,
		Hair:        30000,
		HairColor:   7,
		SkinColor:   0,
		Top:         0,
		Bottom:      0,
		Shoes:       0,
		Weapon:      0,
	}

	// Empty template
	template := template.RestModel{
		JobIndex:    400,
		SubJobIndex: 0,
		MapId:       40000,
		Gender:      0,
		Items:       []uint32{},
		Skills:      []uint32{},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, template)

	// Should only have character creation step
	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result.Steps))
	}

	step := result.Steps[0]
	if step.StepId != "create" {
		t.Errorf("Expected step ID 'create', got '%s'", step.StepId)
	}
	if step.Action != saga.CreateCharacter {
		t.Errorf("Expected action %v, got %v", saga.CreateCharacter, step.Action)
	}
}

func TestBuildCharacterCreationSaga_AllFieldsPresent(t *testing.T) {
	input := RestModel{
		AccountId:   5001,
		WorldId:     255,
		Name:        "CompleteChar",
		Gender:      1,
		JobIndex:    500,
		SubJobIndex: 1,
		Face:        25000,
		Hair:        35000,
		HairColor:   9,
		SkinColor:   4,
		Top:         1040999,
		Bottom:      1060999,
		Shoes:       1072999,
		Weapon:      1302999,
	}

	template := template.RestModel{
		JobIndex:    500,
		SubJobIndex: 1,
		MapId:       50000,
		Gender:      1,
		Items:       []uint32{2000999, 2000998, 2000997},
		Skills:      []uint32{1999, 1998},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, template)

	// Verify character create payload has all fields correctly mapped
	createStep := result.Steps[0]
	if payload, ok := createStep.Payload.(saga.CharacterCreatePayload); ok {
		if payload.AccountId != input.AccountId {
			t.Errorf("AccountId mismatch: expected %d, got %d", input.AccountId, payload.AccountId)
		}
		if payload.Name != input.Name {
			t.Errorf("Name mismatch: expected %s, got %s", input.Name, payload.Name)
		}
		if payload.WorldId != input.WorldId {
			t.Errorf("WorldId mismatch: expected %d, got %d", input.WorldId, payload.WorldId)
		}
		if payload.ChannelId != 0 {
			t.Errorf("ChannelId should default to 0, got %d", payload.ChannelId)
		}
		if payload.JobId != input.JobIndex {
			t.Errorf("JobId mismatch: expected %d, got %d", input.JobIndex, payload.JobId)
		}
		if payload.Face != input.Face {
			t.Errorf("Face mismatch: expected %d, got %d", input.Face, payload.Face)
		}
		if payload.Hair != input.Hair {
			t.Errorf("Hair mismatch: expected %d, got %d", input.Hair, payload.Hair)
		}
		if payload.HairColor != input.HairColor {
			t.Errorf("HairColor mismatch: expected %d, got %d", input.HairColor, payload.HairColor)
		}
		if payload.Skin != uint32(input.SkinColor) {
			t.Errorf("Skin mismatch: expected %d, got %d", uint32(input.SkinColor), payload.Skin)
		}
		if payload.Top != input.Top {
			t.Errorf("Top mismatch: expected %d, got %d", input.Top, payload.Top)
		}
		if payload.Bottom != input.Bottom {
			t.Errorf("Bottom mismatch: expected %d, got %d", input.Bottom, payload.Bottom)
		}
		if payload.Shoes != input.Shoes {
			t.Errorf("Shoes mismatch: expected %d, got %d", input.Shoes, payload.Shoes)
		}
		if payload.Weapon != input.Weapon {
			t.Errorf("Weapon mismatch: expected %d, got %d", input.Weapon, payload.Weapon)
		}
	} else {
		t.Fatal("First step payload is not CharacterCreatePayload")
	}
}

func TestBuildCharacterCreationSaga_Timestamps(t *testing.T) {
	input := RestModel{
		AccountId:   6001,
		WorldId:     0,
		Name:        "TimestampTestChar",
		Gender:      0,
		JobIndex:    100,
		SubJobIndex: 0,
		Face:        20000,
		Hair:        30000,
		HairColor:   7,
		SkinColor:   0,
		Top:         1040002,
		Bottom:      0,
		Shoes:       0,
		Weapon:      0,
	}

	template := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000},
		Skills:      []uint32{},
	}

	beforeTime := time.Now()
	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, template)
	afterTime := time.Now()

	// Verify all steps have proper timestamps
	for i, step := range result.Steps {
		if step.CreatedAt.Before(beforeTime) || step.CreatedAt.After(afterTime) {
			t.Errorf("Step %d CreatedAt timestamp %v is not within expected range [%v, %v]", i, step.CreatedAt, beforeTime, afterTime)
		}
		if step.UpdatedAt.Before(beforeTime) || step.UpdatedAt.After(afterTime) {
			t.Errorf("Step %d UpdatedAt timestamp %v is not within expected range [%v, %v]", i, step.UpdatedAt, beforeTime, afterTime)
		}
		// CreatedAt and UpdatedAt should be the same for newly created steps
		if !step.CreatedAt.Equal(step.UpdatedAt) {
			t.Errorf("Step %d: CreatedAt %v and UpdatedAt %v should be equal for new steps", i, step.CreatedAt, step.UpdatedAt)
		}
	}
}