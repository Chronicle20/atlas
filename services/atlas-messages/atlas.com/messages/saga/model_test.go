package saga

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestSaga_Failing(t *testing.T) {
	testCases := []struct {
		name     string
		steps    []Step[any]
		expected bool
	}{
		{
			name:     "No steps - not failing",
			steps:    []Step[any]{},
			expected: false,
		},
		{
			name: "All pending - not failing",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
				{StepId: "step2", Status: Pending},
			},
			expected: false,
		},
		{
			name: "All completed - not failing",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Completed},
			},
			expected: false,
		},
		{
			name: "One failed - failing",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Failed},
			},
			expected: true,
		},
		{
			name: "First step failed - failing",
			steps: []Step[any]{
				{StepId: "step1", Status: Failed},
				{StepId: "step2", Status: Pending},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga := &Saga{
				TransactionId: uuid.New(),
				Steps:         tc.steps,
			}

			result := saga.Failing()
			if result != tc.expected {
				t.Errorf("Expected Failing()=%v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSaga_GetCurrentStep(t *testing.T) {
	testCases := []struct {
		name          string
		steps         []Step[any]
		expectedId    string
		expectedFound bool
	}{
		{
			name:          "No steps",
			steps:         []Step[any]{},
			expectedId:    "",
			expectedFound: false,
		},
		{
			name: "All completed",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Completed},
			},
			expectedId:    "",
			expectedFound: false,
		},
		{
			name: "First pending",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
				{StepId: "step2", Status: Pending},
			},
			expectedId:    "step1",
			expectedFound: true,
		},
		{
			name: "Second pending",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Pending},
			},
			expectedId:    "step2",
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga := &Saga{
				TransactionId: uuid.New(),
				Steps:         tc.steps,
			}

			step, found := saga.GetCurrentStep()

			if found != tc.expectedFound {
				t.Errorf("Expected found=%v, got %v", tc.expectedFound, found)
			}

			if tc.expectedFound && step.StepId != tc.expectedId {
				t.Errorf("Expected stepId '%s', got '%s'", tc.expectedId, step.StepId)
			}
		})
	}
}

func TestSaga_FindFurthestCompletedStepIndex(t *testing.T) {
	testCases := []struct {
		name     string
		steps    []Step[any]
		expected int
	}{
		{
			name:     "No steps",
			steps:    []Step[any]{},
			expected: -1,
		},
		{
			name: "No completed steps",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
				{StepId: "step2", Status: Pending},
			},
			expected: -1,
		},
		{
			name: "First completed",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Pending},
			},
			expected: 0,
		},
		{
			name: "All completed",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Completed},
				{StepId: "step3", Status: Completed},
			},
			expected: 2,
		},
		{
			name: "Middle completed, last failed",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Completed},
				{StepId: "step3", Status: Failed},
			},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga := &Saga{
				TransactionId: uuid.New(),
				Steps:         tc.steps,
			}

			result := saga.FindFurthestCompletedStepIndex()
			if result != tc.expected {
				t.Errorf("Expected index %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestSaga_FindEarliestPendingStepIndex(t *testing.T) {
	testCases := []struct {
		name     string
		steps    []Step[any]
		expected int
	}{
		{
			name:     "No steps",
			steps:    []Step[any]{},
			expected: -1,
		},
		{
			name: "No pending steps",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Completed},
			},
			expected: -1,
		},
		{
			name: "First pending",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
				{StepId: "step2", Status: Pending},
			},
			expected: 0,
		},
		{
			name: "Second pending",
			steps: []Step[any]{
				{StepId: "step1", Status: Completed},
				{StepId: "step2", Status: Pending},
				{StepId: "step3", Status: Pending},
			},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga := &Saga{
				TransactionId: uuid.New(),
				Steps:         tc.steps,
			}

			result := saga.FindEarliestPendingStepIndex()
			if result != tc.expected {
				t.Errorf("Expected index %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestSaga_SetStepStatus(t *testing.T) {
	testCases := []struct {
		name           string
		steps          []Step[any]
		index          int
		newStatus      Status
		expectedStatus Status
		shouldChange   bool
	}{
		{
			name: "Valid index - change to completed",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
			},
			index:          0,
			newStatus:      Completed,
			expectedStatus: Completed,
			shouldChange:   true,
		},
		{
			name: "Valid index - change to failed",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
			},
			index:          0,
			newStatus:      Failed,
			expectedStatus: Failed,
			shouldChange:   true,
		},
		{
			name: "Invalid negative index",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
			},
			index:          -1,
			newStatus:      Completed,
			expectedStatus: Pending, // Should remain unchanged
			shouldChange:   false,
		},
		{
			name: "Invalid index beyond length",
			steps: []Step[any]{
				{StepId: "step1", Status: Pending},
			},
			index:          5,
			newStatus:      Completed,
			expectedStatus: Pending, // Should remain unchanged
			shouldChange:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saga := &Saga{
				TransactionId: uuid.New(),
				Steps:         tc.steps,
			}

			saga.SetStepStatus(tc.index, tc.newStatus)

			if tc.shouldChange {
				if saga.Steps[tc.index].Status != tc.expectedStatus {
					t.Errorf("Expected status %s, got %s", tc.expectedStatus, saga.Steps[tc.index].Status)
				}
			} else if tc.index >= 0 && tc.index < len(saga.Steps) {
				if saga.Steps[tc.index].Status != tc.expectedStatus {
					t.Errorf("Status should not have changed, expected %s, got %s", tc.expectedStatus, saga.Steps[tc.index].Status)
				}
			}
		})
	}
}

func TestSagaJSON_Marshaling(t *testing.T) {
	transactionId := uuid.New()
	now := time.Now().Truncate(time.Second) // Truncate for JSON comparison

	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "atlas-messages",
		Steps: []Step[any]{
			{
				StepId:    "give_item",
				Status:    Pending,
				Action:    AwardAsset,
				Payload:   AwardItemActionPayload{CharacterId: 12345, Item: ItemPayload{TemplateId: 2000000, Quantity: 1}},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(saga)
	if err != nil {
		t.Fatalf("Failed to marshal saga: %v", err)
	}

	// Verify JSON contains expected fields
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if jsonMap["transactionId"] != transactionId.String() {
		t.Errorf("Expected transactionId %s, got %v", transactionId.String(), jsonMap["transactionId"])
	}

	if jsonMap["sagaType"] != string(InventoryTransaction) {
		t.Errorf("Expected sagaType %s, got %v", InventoryTransaction, jsonMap["sagaType"])
	}

	if jsonMap["initiatedBy"] != "atlas-messages" {
		t.Errorf("Expected initiatedBy 'atlas-messages', got %v", jsonMap["initiatedBy"])
	}

	steps, ok := jsonMap["steps"].([]interface{})
	if !ok || len(steps) != 1 {
		t.Errorf("Expected 1 step in JSON, got %v", jsonMap["steps"])
	}
}

func TestStep_UnmarshalJSON_AwardExperience(t *testing.T) {
	jsonData := `{
		"stepId": "give_exp",
		"status": "pending",
		"action": "award_experience",
		"payload": {
			"characterId": 12345,
			"worldId": 1,
			"channelId": 1,
			"distributions": [{"experienceType": "WHITE", "amount": 1000}]
		},
		"createdAt": "2026-01-13T00:00:00Z",
		"updatedAt": "2026-01-13T00:00:00Z"
	}`

	var step Step[any]
	err := json.Unmarshal([]byte(jsonData), &step)
	if err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	if step.StepId != "give_exp" {
		t.Errorf("Expected stepId 'give_exp', got '%s'", step.StepId)
	}

	if step.Status != Pending {
		t.Errorf("Expected status Pending, got %s", step.Status)
	}

	if step.Action != AwardExperience {
		t.Errorf("Expected action AwardExperience, got %s", step.Action)
	}

	payload, ok := step.Payload.(AwardExperiencePayload)
	if !ok {
		t.Fatalf("Expected payload type AwardExperiencePayload, got %T", step.Payload)
	}

	if payload.CharacterId != 12345 {
		t.Errorf("Expected CharacterId 12345, got %d", payload.CharacterId)
	}

	if payload.WorldId != world.Id(1) {
		t.Errorf("Expected WorldId 1, got %d", payload.WorldId)
	}
}

func TestStep_UnmarshalJSON_AwardAsset(t *testing.T) {
	jsonData := `{
		"stepId": "give_item",
		"status": "pending",
		"action": "award_asset",
		"payload": {
			"characterId": 12345,
			"item": {"templateId": 2000000, "quantity": 10}
		},
		"createdAt": "2026-01-13T00:00:00Z",
		"updatedAt": "2026-01-13T00:00:00Z"
	}`

	var step Step[any]
	err := json.Unmarshal([]byte(jsonData), &step)
	if err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	payload, ok := step.Payload.(AwardItemActionPayload)
	if !ok {
		t.Fatalf("Expected payload type AwardItemActionPayload, got %T", step.Payload)
	}

	if payload.CharacterId != 12345 {
		t.Errorf("Expected CharacterId 12345, got %d", payload.CharacterId)
	}

	if payload.Item.TemplateId != 2000000 {
		t.Errorf("Expected TemplateId 2000000, got %d", payload.Item.TemplateId)
	}

	if payload.Item.Quantity != 10 {
		t.Errorf("Expected Quantity 10, got %d", payload.Item.Quantity)
	}
}

func TestStep_UnmarshalJSON_AwardMesos(t *testing.T) {
	jsonData := `{
		"stepId": "give_mesos",
		"status": "pending",
		"action": "award_mesos",
		"payload": {
			"characterId": 12345,
			"worldId": 1,
			"channelId": 2,
			"actorId": 100,
			"actorType": "CHARACTER",
			"amount": 50000
		},
		"createdAt": "2026-01-13T00:00:00Z",
		"updatedAt": "2026-01-13T00:00:00Z"
	}`

	var step Step[any]
	err := json.Unmarshal([]byte(jsonData), &step)
	if err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	payload, ok := step.Payload.(AwardMesosPayload)
	if !ok {
		t.Fatalf("Expected payload type AwardMesosPayload, got %T", step.Payload)
	}

	if payload.Amount != 50000 {
		t.Errorf("Expected Amount 50000, got %d", payload.Amount)
	}

	if payload.ActorType != "CHARACTER" {
		t.Errorf("Expected ActorType 'CHARACTER', got '%s'", payload.ActorType)
	}
}

func TestStep_UnmarshalJSON_ChangeJob(t *testing.T) {
	jsonData := `{
		"stepId": "change_job",
		"status": "pending",
		"action": "change_job",
		"payload": {
			"characterId": 12345,
			"worldId": 1,
			"channelId": 1,
			"jobId": 100
		},
		"createdAt": "2026-01-13T00:00:00Z",
		"updatedAt": "2026-01-13T00:00:00Z"
	}`

	var step Step[any]
	err := json.Unmarshal([]byte(jsonData), &step)
	if err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	payload, ok := step.Payload.(ChangeJobPayload)
	if !ok {
		t.Fatalf("Expected payload type ChangeJobPayload, got %T", step.Payload)
	}

	if payload.CharacterId != 12345 {
		t.Errorf("Expected CharacterId 12345, got %d", payload.CharacterId)
	}
}

func TestStep_UnmarshalJSON_UnknownAction(t *testing.T) {
	jsonData := `{
		"stepId": "unknown",
		"status": "pending",
		"action": "unknown_action",
		"payload": {},
		"createdAt": "2026-01-13T00:00:00Z",
		"updatedAt": "2026-01-13T00:00:00Z"
	}`

	var step Step[any]
	err := json.Unmarshal([]byte(jsonData), &step)

	if err == nil {
		t.Error("Expected error for unknown action, got nil")
	}
}

func TestStep_UnmarshalJSON_AllActions(t *testing.T) {
	testCases := []struct {
		name       string
		action     Action
		jsonData   string
		payloadType string
	}{
		{
			name:   "AwardLevel",
			action: AwardLevel,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "award_level",
				"payload": {"characterId": 1, "worldId": 1, "channelId": 1, "amount": 5},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "AwardLevelPayload",
		},
		{
			name:   "AwardCurrency",
			action: AwardCurrency,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "award_currency",
				"payload": {"characterId": 1, "accountId": 1, "currencyType": 1, "amount": 100},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "AwardCurrencyPayload",
		},
		{
			name:   "WarpToRandomPortal",
			action: WarpToRandomPortal,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "warp_to_random_portal",
				"payload": {"characterId": 1, "fieldId": "1-1-100000000"},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "WarpToRandomPortalPayload",
		},
		{
			name:   "DestroyAsset",
			action: DestroyAsset,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "destroy_asset",
				"payload": {"characterId": 1, "templateId": 2000000, "quantity": 1},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "DestroyAssetPayload",
		},
		{
			name:   "CreateSkill",
			action: CreateSkill,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "create_skill",
				"payload": {"characterId": 1, "skillId": 1001, "level": 10, "masterLevel": 20, "expiration": "0001-01-01T00:00:00Z"},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "CreateSkillPayload",
		},
		{
			name:   "UpdateSkill",
			action: UpdateSkill,
			jsonData: `{
				"stepId": "test", "status": "pending", "action": "update_skill",
				"payload": {"characterId": 1, "skillId": 1001, "level": 15, "masterLevel": 20, "expiration": "0001-01-01T00:00:00Z"},
				"createdAt": "2026-01-13T00:00:00Z", "updatedAt": "2026-01-13T00:00:00Z"
			}`,
			payloadType: "UpdateSkillPayload",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var step Step[any]
			err := json.Unmarshal([]byte(tc.jsonData), &step)
			if err != nil {
				t.Fatalf("Failed to unmarshal %s: %v", tc.name, err)
			}

			if step.Action != tc.action {
				t.Errorf("Expected action %s, got %s", tc.action, step.Action)
			}

			if step.Payload == nil {
				t.Error("Expected payload to be non-nil")
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if Pending != "pending" {
		t.Errorf("Expected Pending='pending', got '%s'", Pending)
	}

	if Completed != "completed" {
		t.Errorf("Expected Completed='completed', got '%s'", Completed)
	}

	if Failed != "failed" {
		t.Errorf("Expected Failed='failed', got '%s'", Failed)
	}
}

func TestTypeConstants(t *testing.T) {
	if InventoryTransaction != "inventory_transaction" {
		t.Errorf("Expected InventoryTransaction='inventory_transaction', got '%s'", InventoryTransaction)
	}

	if QuestReward != "quest_reward" {
		t.Errorf("Expected QuestReward='quest_reward', got '%s'", QuestReward)
	}

	if TradeTransaction != "trade_transaction" {
		t.Errorf("Expected TradeTransaction='trade_transaction', got '%s'", TradeTransaction)
	}
}

func TestActionConstants(t *testing.T) {
	actions := map[Action]string{
		AwardAsset:         "award_asset",
		AwardExperience:    "award_experience",
		AwardLevel:         "award_level",
		AwardMesos:         "award_mesos",
		AwardCurrency:      "award_currency",
		WarpToRandomPortal: "warp_to_random_portal",
		WarpToPortal:       "warp_to_portal",
		DestroyAsset:       "destroy_asset",
		ChangeJob:          "change_job",
		CreateSkill:        "create_skill",
		UpdateSkill:        "update_skill",
	}

	for action, expected := range actions {
		if string(action) != expected {
			t.Errorf("Expected %s='%s', got '%s'", expected, expected, action)
		}
	}
}
