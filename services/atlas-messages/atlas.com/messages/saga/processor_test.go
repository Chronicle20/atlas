package saga

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestNewProcessor tests the processor constructor
func TestNewProcessor(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	processor := NewProcessor(logger, ctx)

	if processor == nil {
		t.Fatal("NewProcessor() should not return nil")
	}

	// Verify it implements the Processor interface
	var _ = processor
}

// TestProcessorInterface verifies the Processor interface contract
func TestProcessorInterface(t *testing.T) {
	// Verify that ProcessorImpl implements Processor
	var _ Processor = (*ProcessorImpl)(nil)
}

// TestProcessor_CreateWithValidSaga tests saga creation
func TestProcessor_CreateWithValidSaga(t *testing.T) {
	// Note: This test documents the expected behavior
	// Actual execution would require a Kafka connection

	saga, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("atlas-messages").
		AddStep("test_step", Pending, AwardExperience, AwardExperiencePayload{
			CharacterId: 12345,
		}).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Document expected behavior
	t.Logf("Saga TransactionId: %s", saga.TransactionId)
	t.Logf("Saga Type: %s", saga.SagaType)
	t.Logf("Saga InitiatedBy: %s", saga.InitiatedBy)
	t.Logf("Saga Steps: %d", len(saga.Steps))

	// The Create method:
	// 1. Takes a Saga object
	// 2. Calls CreateCommandProvider to create a Kafka message
	// 3. Publishes to the saga command topic via producer.ProviderImpl
}

// TestProcessor_CreateWithMultipleSteps tests saga with multiple steps
func TestProcessor_CreateWithMultipleSteps(t *testing.T) {
	saga, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(QuestReward).
		SetInitiatedBy("COMMAND").
		AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{CharacterId: 1}).
		AddStep("step2", Pending, AwardMesos, AwardMesosPayload{CharacterId: 1, Amount: 1000}).
		AddStep("step3", Pending, AwardInventory, AwardItemActionPayload{CharacterId: 1}).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saga.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(saga.Steps))
	}

	// Document step actions
	expectedActions := []Action{AwardExperience, AwardMesos, AwardInventory}
	for i, step := range saga.Steps {
		if step.Action != expectedActions[i] {
			t.Errorf("Step %d: expected action %s, got %s", i, expectedActions[i], step.Action)
		}
	}
}

// TestProcessor_CreateWithDifferentSagaTypes tests different saga types
func TestProcessor_CreateWithDifferentSagaTypes(t *testing.T) {
	testCases := []struct {
		name     string
		sagaType Type
	}{
		{
			name:     "Inventory transaction",
			sagaType: InventoryTransaction,
		},
		{
			name:     "Quest reward",
			sagaType: QuestReward,
		},
		{
			name:     "Trade transaction",
			sagaType: TradeTransaction,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga, err := NewBuilder().
				SetSagaType(tc.sagaType).
				SetInitiatedBy("test").
				AddStep("test", Pending, AwardExperience, AwardExperiencePayload{}).
				Build()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if saga.SagaType != tc.sagaType {
				t.Errorf("Expected saga type %s, got %s", tc.sagaType, saga.SagaType)
			}
		})
	}
}

// TestProcessor_CreateWithDifferentActions tests all supported actions
func TestProcessor_CreateWithDifferentActions(t *testing.T) {
	testCases := []struct {
		name    string
		action  Action
		payload any
	}{
		{
			name:    "Award Experience",
			action:  AwardExperience,
			payload: AwardExperiencePayload{CharacterId: 12345},
		},
		{
			name:    "Award Level",
			action:  AwardLevel,
			payload: AwardLevelPayload{CharacterId: 12345, Amount: 5},
		},
		{
			name:    "Award Mesos",
			action:  AwardMesos,
			payload: AwardMesosPayload{CharacterId: 12345, Amount: 10000},
		},
		{
			name:    "Award Currency",
			action:  AwardCurrency,
			payload: AwardCurrencyPayload{CharacterId: 12345, CurrencyType: 1, Amount: 100},
		},
		{
			name:    "Award Inventory",
			action:  AwardInventory,
			payload: AwardItemActionPayload{CharacterId: 12345, Item: ItemPayload{TemplateId: 2000000, Quantity: 1}},
		},
		{
			name:    "Change Job",
			action:  ChangeJob,
			payload: ChangeJobPayload{CharacterId: 12345, JobId: 100},
		},
		{
			name:    "Create Skill",
			action:  CreateSkill,
			payload: CreateSkillPayload{CharacterId: 12345, SkillId: 1001, Level: 10},
		},
		{
			name:    "Update Skill",
			action:  UpdateSkill,
			payload: UpdateSkillPayload{CharacterId: 12345, SkillId: 1001, Level: 15},
		},
		{
			name:    "Warp To Random Portal",
			action:  WarpToRandomPortal,
			payload: WarpToRandomPortalPayload{CharacterId: 12345},
		},
		{
			name:    "Destroy Asset",
			action:  DestroyAsset,
			payload: DestroyAssetPayload{CharacterId: 12345, TemplateId: 2000000, Quantity: 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga, err := NewBuilder().
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				AddStep("test_step", Pending, tc.action, tc.payload).
				Build()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(saga.Steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(saga.Steps))
			}

			if saga.Steps[0].Action != tc.action {
				t.Errorf("Expected action %s, got %s", tc.action, saga.Steps[0].Action)
			}
		})
	}
}
