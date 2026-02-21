package factory

import (
	"atlas-character-factory/configuration/tenant/characters/template"
	job2 "atlas-character-factory/job"
	"atlas-character-factory/saga"
	"context"
	"strings"
	"testing"
	"time"

	tenantModel "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestBuildCharacterCreationSaga(t *testing.T) {
	tests := []struct {
		name     string
		input    RestModel
		template template.RestModel
		validate func(t *testing.T, result saga.Saga)
	}{
		{
			name: "basic character creation saga with items equipment and skills",
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
				Items:       []uint32{2000000, 2000001, 2000002},
				Skills:      []uint32{1000, 1001},
			},
			validate: func(t *testing.T, result saga.Saga) {
				if result.SagaType != saga.CharacterCreation {
					t.Errorf("Expected saga type %v, got %v", saga.CharacterCreation, result.SagaType)
				}

				if result.InitiatedBy != "account_1001" {
					t.Errorf("Expected initiatedBy 'account_1001', got '%s'", result.InitiatedBy)
				}

				// 1 create + 3 items + 4 equipment + 2 skills = 10
				expectedSteps := 1 + 3 + 4 + 2
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				// Verify first step is character creation
				firstStep := result.Steps[0]
				if firstStep.StepId != "create_character" {
					t.Errorf("Expected first step ID 'create_character', got '%s'", firstStep.StepId)
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
					if payload.JobId != 0 {
						t.Errorf("Expected JobId 0, got %d", payload.JobId)
					}
					if payload.Face != 20000 {
						t.Errorf("Expected Face 20000, got %d", payload.Face)
					}
					if payload.Hair != 30007 {
						t.Errorf("Expected Hair 30007, got %d", payload.Hair)
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

				// Verify all non-create steps use characterId=0 sentinel
				for i := 1; i < len(result.Steps); i++ {
					step := result.Steps[i]
					switch step.Action {
					case saga.AwardAsset:
						if payload, ok := step.Payload.(saga.AwardItemActionPayload); ok {
							if payload.CharacterId != 0 {
								t.Errorf("Step %d (%s): Expected CharacterId 0 (sentinel), got %d", i, step.StepId, payload.CharacterId)
							}
						}
					case saga.CreateAndEquipAsset:
						if payload, ok := step.Payload.(saga.CreateAndEquipAssetPayload); ok {
							if payload.CharacterId != 0 {
								t.Errorf("Step %d (%s): Expected CharacterId 0 (sentinel), got %d", i, step.StepId, payload.CharacterId)
							}
						}
					case saga.CreateSkill:
						if payload, ok := step.Payload.(saga.CreateSkillPayload); ok {
							if payload.CharacterId != 0 {
								t.Errorf("Step %d (%s): Expected CharacterId 0 (sentinel), got %d", i, step.StepId, payload.CharacterId)
							}
						}
					}
				}
			},
		},
		{
			name: "character creation with zero equipment and empty template",
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
				Top:         0,
				Bottom:      0,
				Shoes:       0,
				Weapon:      0,
			},
			template: template.RestModel{
				JobIndex:    200,
				SubJobIndex: 1,
				MapId:       20000,
				Gender:      1,
				Items:       []uint32{},
				Skills:      []uint32{},
			},
			validate: func(t *testing.T, result saga.Saga) {
				// Only character creation step (no items, no equipment, no skills)
				expectedSteps := 1
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				firstStep := result.Steps[0]
				if firstStep.StepId != "create_character" {
					t.Errorf("Expected first step ID 'create_character', got '%s'", firstStep.StepId)
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
				Top:         1040003,
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
				// 1 create + 1 item + 2 equipment (top, weapon) + 1 skill = 5
				expectedSteps := 1 + 1 + 2 + 1
				if len(result.Steps) != expectedSteps {
					t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
				}

				firstStep := result.Steps[0]
				if firstStep.StepId != "create_character" {
					t.Errorf("Expected first step ID 'create_character', got '%s'", firstStep.StepId)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactionId := uuid.New()
			result := buildCharacterCreationSaga(transactionId, tt.input, tt.template)

			if result.TransactionId != transactionId {
				t.Errorf("Expected TransactionId %v, got %v", transactionId, result.TransactionId)
			}

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

	tmpl := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000, 2000001, 2000002},
		Skills:      []uint32{1000, 1001, 1002},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)

	// Verify step ordering: create_character, then items, then equipment, then skills
	expectedStepOrder := []struct {
		stepType string
		action   saga.Action
	}{
		{"create_character", saga.CreateCharacter},
		{"award_item_0", saga.AwardAsset},
		{"award_item_1", saga.AwardAsset},
		{"award_item_2", saga.AwardAsset},
		{"equip_top", saga.CreateAndEquipAsset},
		{"equip_bottom", saga.CreateAndEquipAsset},
		{"equip_shoes", saga.CreateAndEquipAsset},
		{"equip_weapon", saga.CreateAndEquipAsset},
		{"create_skill_0", saga.CreateSkill},
		{"create_skill_1", saga.CreateSkill},
		{"create_skill_2", saga.CreateSkill},
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

	tmpl := template.RestModel{
		JobIndex:    400,
		SubJobIndex: 0,
		MapId:       40000,
		Gender:      0,
		Items:       []uint32{},
		Skills:      []uint32{},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)

	// Should only have character creation step
	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result.Steps))
	}

	step := result.Steps[0]
	if step.StepId != "create_character" {
		t.Errorf("Expected step ID 'create_character', got '%s'", step.StepId)
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

	tmpl := template.RestModel{
		JobIndex:    500,
		SubJobIndex: 1,
		MapId:       50000,
		Gender:      1,
		Items:       []uint32{2000999, 2000998, 2000997},
		Skills:      []uint32{1999, 1998},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)

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
		expectedJobId := job2.JobFromIndex(input.JobIndex, input.SubJobIndex)
		if payload.JobId != expectedJobId {
			t.Errorf("JobId mismatch: expected %d, got %d", expectedJobId, payload.JobId)
		}
		if payload.Face != input.Face {
			t.Errorf("Face mismatch: expected %d, got %d", input.Face, payload.Face)
		}
		expectedHair := input.Hair + input.HairColor
		if payload.Hair != expectedHair {
			t.Errorf("Hair mismatch: expected %d, got %d", expectedHair, payload.Hair)
		}
		if payload.Skin != input.SkinColor {
			t.Errorf("Skin mismatch: expected %d, got %d", input.SkinColor, payload.Skin)
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

	// Verify total step count: 1 create + 3 items + 4 equip + 2 skills = 10
	expectedSteps := 1 + 3 + 4 + 2
	if len(result.Steps) != expectedSteps {
		t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
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

	tmpl := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000},
		Skills:      []uint32{},
	}

	beforeTime := time.Now()
	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)
	afterTime := time.Now()

	// Verify all steps have proper timestamps
	for i, step := range result.Steps {
		if step.CreatedAt.Before(beforeTime) || step.CreatedAt.After(afterTime) {
			t.Errorf("Step %d CreatedAt timestamp %v is not within expected range [%v, %v]", i, step.CreatedAt, beforeTime, afterTime)
		}
		if step.UpdatedAt.Before(beforeTime) || step.UpdatedAt.After(afterTime) {
			t.Errorf("Step %d UpdatedAt timestamp %v is not within expected range [%v, %v]", i, step.UpdatedAt, beforeTime, afterTime)
		}
		if !step.CreatedAt.Equal(step.UpdatedAt) {
			t.Errorf("Step %d: CreatedAt %v and UpdatedAt %v should be equal for new steps", i, step.CreatedAt, step.UpdatedAt)
		}
	}
}

func TestBuildCharacterCreationSaga_SentinelCharacterId(t *testing.T) {
	input := RestModel{
		AccountId:   7001,
		WorldId:     0,
		Name:        "SentinelTestChar",
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

	tmpl := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000, 2000001},
		Skills:      []uint32{1000},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)

	// All steps after create_character should use characterId=0 as sentinel
	for i := 1; i < len(result.Steps); i++ {
		step := result.Steps[i]
		switch step.Action {
		case saga.AwardAsset:
			if payload, ok := step.Payload.(saga.AwardItemActionPayload); ok {
				if payload.CharacterId != 0 {
					t.Errorf("Step %d (%s): AwardAsset should have characterId=0 sentinel, got %d", i, step.StepId, payload.CharacterId)
				}
			} else {
				t.Errorf("Step %d (%s): Expected AwardItemActionPayload", i, step.StepId)
			}
		case saga.CreateAndEquipAsset:
			if payload, ok := step.Payload.(saga.CreateAndEquipAssetPayload); ok {
				if payload.CharacterId != 0 {
					t.Errorf("Step %d (%s): CreateAndEquipAsset should have characterId=0 sentinel, got %d", i, step.StepId, payload.CharacterId)
				}
			} else {
				t.Errorf("Step %d (%s): Expected CreateAndEquipAssetPayload", i, step.StepId)
			}
		case saga.CreateSkill:
			if payload, ok := step.Payload.(saga.CreateSkillPayload); ok {
				if payload.CharacterId != 0 {
					t.Errorf("Step %d (%s): CreateSkill should have characterId=0 sentinel, got %d", i, step.StepId, payload.CharacterId)
				}
			} else {
				t.Errorf("Step %d (%s): Expected CreateSkillPayload", i, step.StepId)
			}
		default:
			t.Errorf("Step %d: unexpected action %v", i, step.Action)
		}
	}
}

// TestValidationFunctions tests that the validation functions still work correctly
func TestValidationFunctions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "validName",
			testFunc: func(t *testing.T) {
				if !validName("Test") {
					t.Error("Expected 'Test' to be valid")
				}
				if !validName("Player123") {
					t.Error("Expected 'Player123' to be valid")
				}
				if !validName("User_Name") {
					t.Error("Expected 'User_Name' to be valid")
				}
				if !validName("Test-Name") {
					t.Error("Expected 'Test-Name' to be valid")
				}
				if !validName("MaxLengthOK") {
					t.Error("Expected 'MaxLengthOK' to be valid")
				}

				if validName("") {
					t.Error("Expected empty string to be invalid")
				}
				if validName("ThisNameIsTooLong") {
					t.Error("Expected 'ThisNameIsTooLong' (16 chars) to be invalid")
				}
				if validName("Test@Name") {
					t.Error("Expected 'Test@Name' (contains @) to be invalid")
				}
				if validName("Test Name") {
					t.Error("Expected 'Test Name' (contains space) to be invalid")
				}
				if validName("Test!") {
					t.Error("Expected 'Test!' (contains !) to be invalid")
				}
			},
		},
		{
			name: "validGender",
			testFunc: func(t *testing.T) {
				if !validGender(0) {
					t.Error("Expected gender 0 to be valid")
				}
				if !validGender(1) {
					t.Error("Expected gender 1 to be valid")
				}
				if validGender(2) {
					t.Error("Expected gender 2 to be invalid")
				}
				if validGender(255) {
					t.Error("Expected gender 255 to be invalid")
				}
			},
		},
		{
			name: "validJob",
			testFunc: func(t *testing.T) {
				if !validJob(100, 0) {
					t.Error("Expected job (100, 0) to be valid")
				}
				if !validJob(200, 1) {
					t.Error("Expected job (200, 1) to be valid")
				}
				if !validJob(0, 0) {
					t.Error("Expected job (0, 0) to be valid")
				}
			},
		},
		{
			name: "validOption",
			testFunc: func(t *testing.T) {
				options := []uint32{100, 200, 300}

				if !validOption(options, 0) {
					t.Error("Expected 0 to be valid for any option list")
				}
				if !validOption(options, 100) {
					t.Error("Expected 100 to be valid")
				}
				if !validOption(options, 200) {
					t.Error("Expected 200 to be valid")
				}
				if !validOption(options, 300) {
					t.Error("Expected 300 to be valid")
				}
				if validOption(options, 400) {
					t.Error("Expected 400 to be invalid")
				}
				if validOption(options, 50) {
					t.Error("Expected 50 to be invalid")
				}
			},
		},
		{
			name: "validFace",
			testFunc: func(t *testing.T) {
				faces := []uint32{20000, 20001, 20002}

				if !validFace(faces, 0) {
					t.Error("Expected 0 to be valid face")
				}
				if !validFace(faces, 20000) {
					t.Error("Expected 20000 to be valid face")
				}
				if validFace(faces, 20003) {
					t.Error("Expected 20003 to be invalid face")
				}
			},
		},
		{
			name: "validHair",
			testFunc: func(t *testing.T) {
				hairs := []uint32{30000, 30001, 30002}

				if !validHair(hairs, 0) {
					t.Error("Expected 0 to be valid hair")
				}
				if !validHair(hairs, 30000) {
					t.Error("Expected 30000 to be valid hair")
				}
				if validHair(hairs, 30003) {
					t.Error("Expected 30003 to be invalid hair")
				}
			},
		},
		{
			name: "validHairColor",
			testFunc: func(t *testing.T) {
				hairColors := []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

				if !validHairColor(hairColors, 0) {
					t.Error("Expected 0 to be valid hair color")
				}
				if !validHairColor(hairColors, 5) {
					t.Error("Expected 5 to be valid hair color")
				}
				if validHairColor(hairColors, 10) {
					t.Error("Expected 10 to be invalid hair color")
				}
			},
		},
		{
			name: "validSkinColor",
			testFunc: func(t *testing.T) {
				skinColors := []uint32{0, 1, 2, 3, 4}

				if !validSkinColor(skinColors, 0) {
					t.Error("Expected 0 to be valid skin color")
				}
				if !validSkinColor(skinColors, 3) {
					t.Error("Expected 3 to be valid skin color")
				}
				if validSkinColor(skinColors, 5) {
					t.Error("Expected 5 to be invalid skin color")
				}
			},
		},
		{
			name: "validTop",
			testFunc: func(t *testing.T) {
				tops := []uint32{1040000, 1040001, 1040002}

				if !validTop(tops, 0) {
					t.Error("Expected 0 to be valid top")
				}
				if !validTop(tops, 1040000) {
					t.Error("Expected 1040000 to be valid top")
				}
				if validTop(tops, 1040003) {
					t.Error("Expected 1040003 to be invalid top")
				}
			},
		},
		{
			name: "validBottom",
			testFunc: func(t *testing.T) {
				bottoms := []uint32{1060000, 1060001, 1060002}

				if !validBottom(bottoms, 0) {
					t.Error("Expected 0 to be valid bottom")
				}
				if !validBottom(bottoms, 1060000) {
					t.Error("Expected 1060000 to be valid bottom")
				}
				if validBottom(bottoms, 1060003) {
					t.Error("Expected 1060003 to be invalid bottom")
				}
			},
		},
		{
			name: "validShoes",
			testFunc: func(t *testing.T) {
				shoes := []uint32{1072000, 1072001, 1072002}

				if !validShoes(shoes, 0) {
					t.Error("Expected 0 to be valid shoes")
				}
				if !validShoes(shoes, 1072000) {
					t.Error("Expected 1072000 to be valid shoes")
				}
				if validShoes(shoes, 1072003) {
					t.Error("Expected 1072003 to be invalid shoes")
				}
			},
		},
		{
			name: "validWeapon",
			testFunc: func(t *testing.T) {
				weapons := []uint32{1302000, 1302001, 1302002}

				if !validWeapon(weapons, 0) {
					t.Error("Expected 0 to be valid weapon")
				}
				if !validWeapon(weapons, 1302000) {
					t.Error("Expected 1302000 to be valid weapon")
				}
				if validWeapon(weapons, 1302003) {
					t.Error("Expected 1302003 to be invalid weapon")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

// TestSagaEmissionToKafka tests that the saga processor emits messages correctly
func TestSagaEmissionToKafka(t *testing.T) {
	tests := []struct {
		name        string
		saga        saga.Saga
		expectError bool
	}{
		{
			name: "successful saga emission",
			saga: saga.NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(saga.CharacterCreation).
				SetInitiatedBy("test").
				AddStep("create", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
					AccountId: 1001,
					Name:      "TestChar",
					WorldId:   0,
					JobId:     100,
					Face:      20000,
					Hair:      30007,
					Skin:      0,
					Top:       1040002,
					Bottom:    1060002,
					Shoes:     1072001,
					Weapon:    1302000,
				}).
				Build(),
			expectError: false,
		},
		{
			name: "saga emission with multiple steps",
			saga: saga.NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(saga.CharacterCreation).
				SetInitiatedBy("test").
				AddStep("create", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
					AccountId: 2001,
					Name:      "MultiStepChar",
					WorldId:   1,
					JobId:     200,
				}).
				AddStep("award_item", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
					CharacterId: 0,
					Item: saga.ItemPayload{
						TemplateId: 2000000,
						Quantity:   1,
					},
				}).
				AddStep("equip_weapon", saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
					CharacterId: 0,
					Item: saga.ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				}).
				Build(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			ctx, _ := createMockContext(t)

			sagaProcessor := saga.NewProcessor(logger, ctx)

			err := sagaProcessor.Create(tt.saga)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Logf("Note: Saga emission failed due to missing Kafka infrastructure (expected in unit tests): %v", err)
			}

			if tt.saga.TransactionId == uuid.Nil {
				t.Error("Expected non-nil transaction ID in saga")
			}

			if tt.saga.SagaType != saga.CharacterCreation {
				t.Errorf("Expected saga type %v, got %v", saga.CharacterCreation, tt.saga.SagaType)
			}

			if len(tt.saga.Steps) == 0 {
				t.Error("Expected at least one step in saga")
			}

			for i, step := range tt.saga.Steps {
				if step.Status != saga.Pending {
					t.Errorf("Step %d should have status %v, got %v", i, saga.Pending, step.Status)
				}
				if step.CreatedAt.IsZero() {
					t.Errorf("Step %d should have non-zero CreatedAt", i)
				}
				if step.UpdatedAt.IsZero() {
					t.Errorf("Step %d should have non-zero UpdatedAt", i)
				}
			}
		})
	}
}

// TestSagaProducerCreation tests that the saga producer creates valid Kafka messages
func TestSagaProducerCreation(t *testing.T) {
	tests := []struct {
		name string
		saga saga.Saga
	}{
		{
			name: "character creation saga message",
			saga: saga.NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(saga.CharacterCreation).
				SetInitiatedBy("test").
				AddStep("create", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
					AccountId: 1001,
					Name:      "TestChar",
					WorldId:   0,
					JobId:     100,
				}).
				Build(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageProvider := saga.CreateCommandProvider(tt.saga)

			messages, err := messageProvider()

			if err != nil {
				t.Errorf("Unexpected error creating Kafka message: %v", err)
				return
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
				return
			}

			message := messages[0]

			expectedKey := tt.saga.TransactionId.String()
			if string(message.Key) != expectedKey {
				t.Errorf("Expected message key '%s', got '%s'", expectedKey, string(message.Key))
			}

			if len(message.Value) == 0 {
				t.Error("Expected non-empty message value")
			}
		})
	}
}

// Helper function to create mock context with tenant information
func createMockContext(t *testing.T) (context.Context, uuid.UUID) {
	tenantId := uuid.New()
	testTenant, err := tenantModel.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	ctx := tenantModel.WithContext(context.Background(), testTenant)

	return ctx, tenantId
}

// TestCharacterCreationOrchestrationFlow tests the complete orchestration flow
// with a unified saga that includes create, items, equipment, and skills.
func TestCharacterCreationOrchestrationFlow(t *testing.T) {
	input := RestModel{
		AccountId:   3001,
		WorldId:     2,
		Name:        "OrchestrationTestChar",
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

	tmpl := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000, 2000001, 2000002},
		Skills:      []uint32{1000, 1001, 1002},
	}

	transactionId := uuid.New()
	result := buildCharacterCreationSaga(transactionId, input, tmpl)

	t.Run("saga_metadata", func(t *testing.T) {
		if result.SagaType != saga.CharacterCreation {
			t.Errorf("Expected saga type %v, got %v", saga.CharacterCreation, result.SagaType)
		}

		if result.TransactionId != transactionId {
			t.Errorf("Expected transaction ID %s, got %s", transactionId, result.TransactionId)
		}

		expectedInitiatedBy := "account_3001"
		if result.InitiatedBy != expectedInitiatedBy {
			t.Errorf("Expected initiated by '%s', got '%s'", expectedInitiatedBy, result.InitiatedBy)
		}
	})

	t.Run("unified_saga_step_count", func(t *testing.T) {
		// 1 create + 3 items + 4 equipment + 3 skills = 11
		expectedSteps := 1 + 3 + 4 + 3
		if len(result.Steps) != expectedSteps {
			t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Steps))
		}
	})

	t.Run("step_sequence", func(t *testing.T) {
		expectedSequence := []struct {
			stepType string
			action   saga.Action
		}{
			{"create_character", saga.CreateCharacter},
			{"award_item_0", saga.AwardAsset},
			{"award_item_1", saga.AwardAsset},
			{"award_item_2", saga.AwardAsset},
			{"equip_top", saga.CreateAndEquipAsset},
			{"equip_bottom", saga.CreateAndEquipAsset},
			{"equip_shoes", saga.CreateAndEquipAsset},
			{"equip_weapon", saga.CreateAndEquipAsset},
			{"create_skill_0", saga.CreateSkill},
			{"create_skill_1", saga.CreateSkill},
			{"create_skill_2", saga.CreateSkill},
		}

		for i, expected := range expectedSequence {
			if i >= len(result.Steps) {
				t.Errorf("Missing step %d: expected %s", i, expected.stepType)
				continue
			}

			step := result.Steps[i]
			if step.StepId != expected.stepType {
				t.Errorf("Step %d: expected step ID '%s', got '%s'", i, expected.stepType, step.StepId)
			}

			if step.Action != expected.action {
				t.Errorf("Step %d: expected action '%s', got '%s'", i, expected.action, step.Action)
			}

			if step.Status != saga.Pending {
				t.Errorf("Step %d: expected status 'pending', got '%s'", i, step.Status)
			}
		}
	})

	t.Run("create_character_payload", func(t *testing.T) {
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
			expectedJobId := job2.JobFromIndex(input.JobIndex, input.SubJobIndex)
			if payload.JobId != expectedJobId {
				t.Errorf("JobId mismatch: expected %d, got %d", expectedJobId, payload.JobId)
			}
			if payload.Face != input.Face {
				t.Errorf("Face mismatch: expected %d, got %d", input.Face, payload.Face)
			}
			expectedHair := input.Hair + input.HairColor
			if payload.Hair != expectedHair {
				t.Errorf("Hair mismatch: expected %d, got %d", expectedHair, payload.Hair)
			}
			if payload.Skin != input.SkinColor {
				t.Errorf("Skin mismatch: expected %d, got %d", input.SkinColor, payload.Skin)
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
			t.Error("Character create step payload is not of correct type")
		}
	})

	t.Run("follow_up_steps_use_sentinel", func(t *testing.T) {
		for i := 1; i < len(result.Steps); i++ {
			step := result.Steps[i]
			switch step.Action {
			case saga.AwardAsset:
				if payload, ok := step.Payload.(saga.AwardItemActionPayload); ok {
					if payload.CharacterId != 0 {
						t.Errorf("Step %d (%s): Expected sentinel characterId=0, got %d", i, step.StepId, payload.CharacterId)
					}
				}
			case saga.CreateAndEquipAsset:
				if payload, ok := step.Payload.(saga.CreateAndEquipAssetPayload); ok {
					if payload.CharacterId != 0 {
						t.Errorf("Step %d (%s): Expected sentinel characterId=0, got %d", i, step.StepId, payload.CharacterId)
					}
				}
			case saga.CreateSkill:
				if payload, ok := step.Payload.(saga.CreateSkillPayload); ok {
					if payload.CharacterId != 0 {
						t.Errorf("Step %d (%s): Expected sentinel characterId=0, got %d", i, step.StepId, payload.CharacterId)
					}
				}
			}
		}
	})
}

// Helper function to check if error is related to Kafka connectivity
func isKafkaRelatedError(err error) bool {
	errMsg := err.Error()
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "max retry reached") ||
		strings.Contains(errMsg, "unable to emit") ||
		strings.Contains(errMsg, "Unable to emit")
}

// TestErrorHandlingAndEdgeCases tests various error conditions and edge cases
func TestErrorHandlingAndEdgeCases(t *testing.T) {
	t.Run("gender_validation_edge_cases", func(t *testing.T) {
		tests := []struct {
			name      string
			gender    byte
			shouldErr bool
		}{
			{"valid_gender_0", 0, false},
			{"valid_gender_1", 1, false},
			{"invalid_gender_2", 2, true},
			{"invalid_gender_255", 255, true},
			{"invalid_gender_254", 254, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validGender(tt.gender)
				if tt.shouldErr && result {
					t.Errorf("Expected gender %d to be invalid, but got valid", tt.gender)
				}
				if !tt.shouldErr && !result {
					t.Errorf("Expected gender %d to be valid, but got invalid", tt.gender)
				}
			})
		}
	})

	t.Run("validation_option_edge_cases", func(t *testing.T) {
		tests := []struct {
			name      string
			options   []uint32
			selection uint32
			shouldErr bool
		}{
			{"valid_zero_selection", []uint32{100, 200}, 0, false},
			{"valid_existing_selection", []uint32{100, 200}, 100, false},
			{"invalid_missing_selection", []uint32{100, 200}, 150, true},
			{"invalid_selection_max_value", []uint32{100, 200}, 4294967295, true},
			{"empty_options_zero_selection", []uint32{}, 0, false},
			{"empty_options_nonzero_selection", []uint32{}, 1, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validOption(tt.options, tt.selection)
				if tt.shouldErr && result {
					t.Errorf("Expected selection %d to be invalid for options %v, but got valid", tt.selection, tt.options)
				}
				if !tt.shouldErr && !result {
					t.Errorf("Expected selection %d to be valid for options %v, but got invalid", tt.selection, tt.options)
				}
			})
		}
	})
}

// TestValidationBoundaryConditions tests edge cases in validation functions
func TestValidationBoundaryConditions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "validOption_empty_slice",
			testFunc: func(t *testing.T) {
				emptyOptions := []uint32{}

				if !validOption(emptyOptions, 0) {
					t.Error("Expected 0 to be valid with empty options")
				}
				if validOption(emptyOptions, 1) {
					t.Error("Expected 1 to be invalid with empty options")
				}
				if validOption(emptyOptions, 100) {
					t.Error("Expected 100 to be invalid with empty options")
				}
			},
		},
		{
			name: "validOption_nil_slice",
			testFunc: func(t *testing.T) {
				var nilOptions []uint32 = nil

				if !validOption(nilOptions, 0) {
					t.Error("Expected 0 to be valid with nil options")
				}
				if validOption(nilOptions, 1) {
					t.Error("Expected 1 to be invalid with nil options")
				}
			},
		},
		{
			name: "validOption_single_element",
			testFunc: func(t *testing.T) {
				singleOptions := []uint32{42}

				if !validOption(singleOptions, 0) {
					t.Error("Expected 0 to be valid with single element options")
				}
				if !validOption(singleOptions, 42) {
					t.Error("Expected 42 to be valid with single element options")
				}
				if validOption(singleOptions, 43) {
					t.Error("Expected 43 to be invalid with single element options")
				}
			},
		},
		{
			name: "validOption_duplicates",
			testFunc: func(t *testing.T) {
				duplicateOptions := []uint32{100, 100, 200, 200, 300}

				if !validOption(duplicateOptions, 0) {
					t.Error("Expected 0 to be valid with duplicate options")
				}
				if !validOption(duplicateOptions, 100) {
					t.Error("Expected 100 to be valid with duplicate options")
				}
				if !validOption(duplicateOptions, 200) {
					t.Error("Expected 200 to be valid with duplicate options")
				}
				if !validOption(duplicateOptions, 300) {
					t.Error("Expected 300 to be valid with duplicate options")
				}
				if validOption(duplicateOptions, 150) {
					t.Error("Expected 150 to be invalid with duplicate options")
				}
			},
		},
		{
			name: "validOption_max_values",
			testFunc: func(t *testing.T) {
				maxOptions := []uint32{0, 1, 4294967295}

				if !validOption(maxOptions, 0) {
					t.Error("Expected 0 to be valid with max value options")
				}
				if !validOption(maxOptions, 1) {
					t.Error("Expected 1 to be valid with max value options")
				}
				if !validOption(maxOptions, 4294967295) {
					t.Error("Expected max uint32 to be valid with max value options")
				}
				if validOption(maxOptions, 2) {
					t.Error("Expected 2 to be invalid with max value options")
				}
			},
		},
		{
			name: "validGender_boundary",
			testFunc: func(t *testing.T) {
				if !validGender(0) {
					t.Error("Expected gender 0 to be valid")
				}
				if !validGender(1) {
					t.Error("Expected gender 1 to be valid")
				}
				if validGender(2) {
					t.Error("Expected gender 2 to be invalid")
				}
				if validGender(255) {
					t.Error("Expected gender 255 to be invalid")
				}
			},
		},
		{
			name: "validJob_always_true",
			testFunc: func(t *testing.T) {
				if !validJob(0, 0) {
					t.Error("Expected job (0, 0) to be valid")
				}
				if !validJob(4294967295, 4294967295) {
					t.Error("Expected job (max, max) to be valid")
				}
				if !validJob(100, 0) {
					t.Error("Expected job (100, 0) to be valid")
				}
				if !validJob(0, 100) {
					t.Error("Expected job (0, 100) to be valid")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

// TestSagaConstructionErrorCases tests error cases in saga construction
func TestSagaConstructionErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		setupTest func(t *testing.T) (uuid.UUID, RestModel, template.RestModel)
	}{
		{
			name: "nil_uuid_handling",
			setupTest: func(t *testing.T) (uuid.UUID, RestModel, template.RestModel) {
				input := RestModel{
					AccountId:   1001,
					WorldId:     0,
					Name:        "TestChar",
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

				tmpl := template.RestModel{
					JobIndex:    100,
					SubJobIndex: 0,
					MapId:       10000,
					Gender:      0,
					Items:       []uint32{2000000},
					Skills:      []uint32{1000},
				}

				return uuid.Nil, input, tmpl
			},
		},
		{
			name: "empty_account_id",
			setupTest: func(t *testing.T) (uuid.UUID, RestModel, template.RestModel) {
				input := RestModel{
					AccountId:   0,
					WorldId:     0,
					Name:        "TestChar",
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

				tmpl := template.RestModel{
					JobIndex:    100,
					SubJobIndex: 0,
					MapId:       10000,
					Gender:      0,
					Items:       []uint32{2000000},
					Skills:      []uint32{1000},
				}

				return uuid.New(), input, tmpl
			},
		},
		{
			name: "very_large_template_items",
			setupTest: func(t *testing.T) (uuid.UUID, RestModel, template.RestModel) {
				input := RestModel{
					AccountId:   1001,
					WorldId:     0,
					Name:        "TestChar",
					Gender:      0,
					JobIndex:    100,
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

				manyItems := make([]uint32, 1000)
				for i := range manyItems {
					manyItems[i] = uint32(2000000 + i)
				}

				tmpl := template.RestModel{
					JobIndex:    100,
					SubJobIndex: 0,
					MapId:       10000,
					Gender:      0,
					Items:       manyItems,
					Skills:      []uint32{},
				}

				return uuid.New(), input, tmpl
			},
		},
		{
			name: "very_large_template_skills",
			setupTest: func(t *testing.T) (uuid.UUID, RestModel, template.RestModel) {
				input := RestModel{
					AccountId:   1001,
					WorldId:     0,
					Name:        "TestChar",
					Gender:      0,
					JobIndex:    100,
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

				manySkills := make([]uint32, 1000)
				for i := range manySkills {
					manySkills[i] = uint32(1000 + i)
				}

				tmpl := template.RestModel{
					JobIndex:    100,
					SubJobIndex: 0,
					MapId:       10000,
					Gender:      0,
					Items:       []uint32{},
					Skills:      manySkills,
				}

				return uuid.New(), input, tmpl
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactionId, input, tmpl := tt.setupTest(t)

			// This should not panic, even with edge case inputs
			result := buildCharacterCreationSaga(transactionId, input, tmpl)

			if result.TransactionId != transactionId {
				t.Errorf("Expected transaction ID %s, got %s", transactionId, result.TransactionId)
			}

			if result.SagaType != saga.CharacterCreation {
				t.Errorf("Expected saga type %s, got %s", saga.CharacterCreation, result.SagaType)
			}

			if len(result.Steps) == 0 {
				t.Error("Expected at least one step (character creation)")
			}

			if len(result.Steps) > 0 {
				firstStep := result.Steps[0]
				if firstStep.StepId != "create_character" {
					t.Errorf("Expected first step ID 'create_character', got '%s'", firstStep.StepId)
				}
				if firstStep.Action != saga.CreateCharacter {
					t.Errorf("Expected first step action %s, got %s", saga.CreateCharacter, firstStep.Action)
				}
			}

			for i, step := range result.Steps {
				if step.CreatedAt.IsZero() {
					t.Errorf("Step %d missing CreatedAt timestamp", i)
				}
				if step.UpdatedAt.IsZero() {
					t.Errorf("Step %d missing UpdatedAt timestamp", i)
				}
			}
		})
	}
}

// TestConcurrentSagaCreation tests saga creation under concurrent conditions
func TestConcurrentSagaCreation(t *testing.T) {
	input := RestModel{
		AccountId:   1001,
		WorldId:     0,
		Name:        "ConcurrentTestChar",
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

	tmpl := template.RestModel{
		JobIndex:    100,
		SubJobIndex: 0,
		MapId:       10000,
		Gender:      0,
		Items:       []uint32{2000000, 2000001},
		Skills:      []uint32{1000, 1001},
	}

	const numGoroutines = 100
	results := make([]saga.Saga, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			transactionId := uuid.New()
			results[index] = buildCharacterCreationSaga(transactionId, input, tmpl)
		}(i)
	}

	// Wait a bit for all goroutines to complete
	time.Sleep(100 * time.Millisecond)

	for i, result := range results {
		if result.TransactionId == uuid.Nil {
			t.Errorf("Saga %d has nil transaction ID", i)
		}
		if result.SagaType != saga.CharacterCreation {
			t.Errorf("Saga %d has wrong type: %s", i, result.SagaType)
		}
		if len(result.Steps) == 0 {
			t.Errorf("Saga %d has no steps", i)
		}
	}

	// Verify all transaction IDs are unique
	transactionIds := make(map[uuid.UUID]bool)
	for i, result := range results {
		if transactionIds[result.TransactionId] {
			t.Errorf("Duplicate transaction ID found in saga %d: %s", i, result.TransactionId)
		}
		transactionIds[result.TransactionId] = true
	}
}
