package saga

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestNewBuilder(t *testing.T) {
	b := NewBuilder()

	if b == nil {
		t.Fatal("NewBuilder() should not return nil")
	}

	if b.transactionId == uuid.Nil {
		t.Error("NewBuilder() should initialize transactionId with a valid UUID")
	}

	if b.steps == nil {
		t.Error("NewBuilder() should initialize steps slice")
	}

	if len(b.steps) != 0 {
		t.Errorf("NewBuilder() should initialize empty steps slice, got %d steps", len(b.steps))
	}
}

func TestBuilder_SetTransactionId(t *testing.T) {
	b := NewBuilder()
	customId := uuid.New()

	result := b.SetTransactionId(customId)

	// Verify fluent API returns same builder
	if result != b {
		t.Error("SetTransactionId should return the same builder instance")
	}

	if b.transactionId != customId {
		t.Errorf("Expected transactionId %v, got %v", customId, b.transactionId)
	}
}

func TestBuilder_SetSagaType(t *testing.T) {
	testCases := []struct {
		name     string
		sagaType Type
	}{
		{
			name:     "InventoryTransaction type",
			sagaType: InventoryTransaction,
		},
		{
			name:     "QuestReward type",
			sagaType: QuestReward,
		},
		{
			name:     "TradeTransaction type",
			sagaType: TradeTransaction,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			result := b.SetSagaType(tc.sagaType)

			if result != b {
				t.Error("SetSagaType should return the same builder instance")
			}

			if b.sagaType != tc.sagaType {
				t.Errorf("Expected sagaType %v, got %v", tc.sagaType, b.sagaType)
			}
		})
	}
}

func TestBuilder_SetInitiatedBy(t *testing.T) {
	testCases := []struct {
		name        string
		initiatedBy string
	}{
		{
			name:        "atlas-messages initiator",
			initiatedBy: "atlas-messages",
		},
		{
			name:        "COMMAND initiator",
			initiatedBy: "COMMAND",
		},
		{
			name:        "NPC initiator",
			initiatedBy: "NPC_1234",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			result := b.SetInitiatedBy(tc.initiatedBy)

			if result != b {
				t.Error("SetInitiatedBy should return the same builder instance")
			}

			if b.initiatedBy != tc.initiatedBy {
				t.Errorf("Expected initiatedBy '%s', got '%s'", tc.initiatedBy, b.initiatedBy)
			}
		})
	}
}

func TestBuilder_AddStep(t *testing.T) {
	b := NewBuilder()

	result := b.AddStep("test_step", Pending, AwardExperience, AwardExperiencePayload{
		CharacterId: 12345,
		WorldId:     world.Id(1),
		ChannelId:   channel.Id(1),
		Distributions: []ExperienceDistributions{{
			ExperienceType: "WHITE",
			Amount:         1000,
		}},
	})

	if result != b {
		t.Error("AddStep should return the same builder instance")
	}

	if len(b.steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(b.steps))
	}

	step := b.steps[0]
	if step.StepId != "test_step" {
		t.Errorf("Expected stepId 'test_step', got '%s'", step.StepId)
	}

	if step.Status != Pending {
		t.Errorf("Expected status Pending, got %s", step.Status)
	}

	if step.Action != AwardExperience {
		t.Errorf("Expected action AwardExperience, got %s", step.Action)
	}

	if step.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if step.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestBuilder_AddStep_MultipleSteps(t *testing.T) {
	b := NewBuilder()

	b.AddStep("step_1", Pending, AwardExperience, AwardExperiencePayload{CharacterId: 1})
	b.AddStep("step_2", Pending, AwardLevel, AwardLevelPayload{CharacterId: 2, Amount: 5})
	b.AddStep("step_3", Pending, AwardMesos, AwardMesosPayload{CharacterId: 3, Amount: 1000})

	if len(b.steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(b.steps))
	}

	expectedIds := []string{"step_1", "step_2", "step_3"}
	for i, expectedId := range expectedIds {
		if b.steps[i].StepId != expectedId {
			t.Errorf("Step %d: expected stepId '%s', got '%s'", i, expectedId, b.steps[i].StepId)
		}
	}
}

func TestBuilder_AddStep_DifferentPayloadTypes(t *testing.T) {
	testCases := []struct {
		name    string
		action  Action
		payload any
	}{
		{
			name:   "AwardExperience payload",
			action: AwardExperience,
			payload: AwardExperiencePayload{
				CharacterId: 12345,
				WorldId:     world.Id(1),
				ChannelId:   channel.Id(1),
			},
		},
		{
			name:   "AwardLevel payload",
			action: AwardLevel,
			payload: AwardLevelPayload{
				CharacterId: 12345,
				Amount:      10,
			},
		},
		{
			name:   "AwardMesos payload",
			action: AwardMesos,
			payload: AwardMesosPayload{
				CharacterId: 12345,
				Amount:      50000,
				ActorType:   "CHARACTER",
			},
		},
		{
			name:   "AwardInventory payload",
			action: AwardInventory,
			payload: AwardItemActionPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 2000000,
					Quantity:   10,
				},
			},
		},
		{
			name:   "ChangeJob payload",
			action: ChangeJob,
			payload: ChangeJobPayload{
				CharacterId: 12345,
				JobId:       100,
			},
		},
		{
			name:   "CreateSkill payload",
			action: CreateSkill,
			payload: CreateSkillPayload{
				CharacterId: 12345,
				SkillId:     1001,
				Level:       10,
				MasterLevel: 20,
				Expiration:  time.Time{},
			},
		},
		{
			name:   "UpdateSkill payload",
			action: UpdateSkill,
			payload: UpdateSkillPayload{
				CharacterId: 12345,
				SkillId:     1001,
				Level:       15,
				MasterLevel: 20,
			},
		},
		{
			name:   "AwardCurrency payload",
			action: AwardCurrency,
			payload: AwardCurrencyPayload{
				CharacterId:  12345,
				AccountId:    100,
				CurrencyType: 1,
				Amount:       500,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			b.AddStep("test_step", Pending, tc.action, tc.payload)

			if len(b.steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(b.steps))
			}

			if b.steps[0].Action != tc.action {
				t.Errorf("Expected action %s, got %s", tc.action, b.steps[0].Action)
			}

			if b.steps[0].Payload == nil {
				t.Error("Expected payload to be set")
			}
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	customId := uuid.New()

	saga, err := NewBuilder().
		SetTransactionId(customId).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("atlas-messages").
		AddStep("give_item", Pending, AwardInventory, AwardItemActionPayload{
			CharacterId: 12345,
			Item: ItemPayload{
				TemplateId: 2000000,
				Quantity:   1,
			},
		}).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if saga.TransactionId != customId {
		t.Errorf("Expected TransactionId %v, got %v", customId, saga.TransactionId)
	}

	if saga.SagaType != InventoryTransaction {
		t.Errorf("Expected SagaType %s, got %s", InventoryTransaction, saga.SagaType)
	}

	if saga.InitiatedBy != "atlas-messages" {
		t.Errorf("Expected InitiatedBy 'atlas-messages', got '%s'", saga.InitiatedBy)
	}

	if len(saga.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(saga.Steps))
	}

	step := saga.Steps[0]
	if step.StepId != "give_item" {
		t.Errorf("Expected stepId 'give_item', got '%s'", step.StepId)
	}
}

func TestBuilder_Build_EmptySteps(t *testing.T) {
	_, err := NewBuilder().
		SetSagaType(QuestReward).
		SetInitiatedBy("COMMAND").
		Build()

	if err == nil {
		t.Fatal("expected error for empty steps, got nil")
	}

	expectedMsg := "saga must have at least one step"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBuilder_FluentAPI(t *testing.T) {
	// Verify the entire fluent API chain works correctly
	b := NewBuilder()

	result := b.
		SetTransactionId(uuid.New()).
		SetSagaType(TradeTransaction).
		SetInitiatedBy("test").
		AddStep("step1", Pending, AwardMesos, AwardMesosPayload{}).
		AddStep("step2", Pending, AwardExperience, AwardExperiencePayload{})

	// All methods should return the same builder
	if result != b {
		t.Error("Fluent API chain should return the same builder instance")
	}

	saga, err := result.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saga.Steps) != 2 {
		t.Errorf("Expected 2 steps from fluent chain, got %d", len(saga.Steps))
	}
}

func TestBuilder_StepTimestamps(t *testing.T) {
	before := time.Now()
	b := NewBuilder().AddStep("test", Pending, AwardExperience, AwardExperiencePayload{})
	after := time.Now()

	step := b.steps[0]

	if step.CreatedAt.Before(before) || step.CreatedAt.After(after) {
		t.Error("CreatedAt should be set to current time")
	}

	if step.UpdatedAt.Before(before) || step.UpdatedAt.After(after) {
		t.Error("UpdatedAt should be set to current time")
	}

	if !step.CreatedAt.Equal(step.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal for new steps")
	}
}

func TestBuilder_Build_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		builder     func() *Builder
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid builder succeeds",
			builder: func() *Builder {
				return NewBuilder().
					SetSagaType(QuestReward).
					SetInitiatedBy("COMMAND").
					AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{})
			},
			expectError: false,
		},
		{
			name: "no saga type fails",
			builder: func() *Builder {
				return NewBuilder().
					SetInitiatedBy("COMMAND").
					AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{})
			},
			expectError: true,
			errorMsg:    "saga type is required",
		},
		{
			name: "no initiatedBy fails",
			builder: func() *Builder {
				return NewBuilder().
					SetSagaType(QuestReward).
					AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{})
			},
			expectError: true,
			errorMsg:    "initiatedBy is required",
		},
		{
			name: "empty action fails",
			builder: func() *Builder {
				return NewBuilder().
					SetSagaType(QuestReward).
					SetInitiatedBy("COMMAND").
					AddStep("step1", Pending, "", AwardExperiencePayload{})
			},
			expectError: true,
			errorMsg:    "step 0 has invalid action",
		},
		{
			name: "multiple steps with one invalid action fails",
			builder: func() *Builder {
				return NewBuilder().
					SetSagaType(QuestReward).
					SetInitiatedBy("COMMAND").
					AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{}).
					AddStep("step2", Pending, "", AwardLevelPayload{})
			},
			expectError: true,
			errorMsg:    "step 1 has invalid action",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.builder().Build()
			if tc.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tc.errorMsg {
					t.Errorf("expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
