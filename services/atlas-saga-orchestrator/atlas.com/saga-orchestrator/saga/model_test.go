package saga

import (
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

func TestSaga_Failing(t *testing.T) {
	tests := []struct {
		name     string
		steps    []Step[any]
		expected bool
	}{
		{
			name:     "No steps",
			steps:    []Step[any]{},
			expected: false,
		},
		{
			name: "No failing steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: false,
		},
		{
			name: "With failing step",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Failed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				Build()
			
			saga.Steps = tt.steps
			
			if got := saga.Failing(); got != tt.expected {
				t.Errorf("Saga.Failing() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSaga_GetCurrentStep(t *testing.T) {
	tests := []struct {
		name          string
		steps         []Step[any]
		expectStep    bool
		expectedIndex int
	}{
		{
			name:          "No steps",
			steps:         []Step[any]{},
			expectStep:    false,
			expectedIndex: -1,
		},
		{
			name: "No pending steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expectStep:    false,
			expectedIndex: -1,
		},
		{
			name: "With pending step",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step3",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expectStep:    true,
			expectedIndex: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				Build()
			
			saga.Steps = tt.steps
			
			step, found := saga.GetCurrentStep()
			if found != tt.expectStep {
				t.Errorf("Saga.GetCurrentStep() found = %v, want %v", found, tt.expectStep)
			}
			
			if found && step.StepId != tt.steps[tt.expectedIndex].StepId {
				t.Errorf("Saga.GetCurrentStep() returned step with ID = %v, want step with ID = %v", 
					step.StepId, tt.steps[tt.expectedIndex].StepId)
			}
		})
	}
}

func TestSaga_FindFurthestCompletedStepIndex(t *testing.T) {
	tests := []struct {
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
				{
					StepId:    "step1",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: -1,
		},
		{
			name: "With completed steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step3",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "With mixed status steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Failed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step3",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step4",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				Build()
			
			saga.Steps = tt.steps
			
			if got := saga.FindFurthestCompletedStepIndex(); got != tt.expected {
				t.Errorf("Saga.FindFurthestCompletedStepIndex() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSaga_FindEarliestPendingStepIndex(t *testing.T) {
	tests := []struct {
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
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: -1,
		},
		{
			name: "With pending steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step3",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "With mixed status steps",
			steps: []Step[any]{
				{
					StepId:    "step1",
					Status:    Failed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step2",
					Status:    Completed,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step3",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					StepId:    "step4",
					Status:    Pending,
					Action:    AwardInventory,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := NewBuilder().
				SetTransactionId(uuid.New()).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				Build()
			
			saga.Steps = tt.steps
			
			if got := saga.FindEarliestPendingStepIndex(); got != tt.expected {
				t.Errorf("Saga.FindEarliestPendingStepIndex() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilder(t *testing.T) {
	// Test that the builder correctly constructs a Saga
	transactionId := uuid.New()
	sagaType := InventoryTransaction
	initiatedBy := "test-initiator"
	
	builder := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sagaType).
		SetInitiatedBy(initiatedBy)
	
	// Add some steps
	payload := AwardItemActionPayload{
		CharacterId: 12345,
		Item: ItemPayload{
			TemplateId: 67890,
			Quantity:   5,
		},
	}
	
	builder.AddStep("step1", Pending, AwardInventory, payload)
	builder.AddStep("step2", Completed, AwardInventory, payload)
	
	// Build the saga
	saga := builder.Build()
	
	// Verify the saga properties
	if saga.TransactionId != transactionId {
		t.Errorf("Builder set TransactionId = %v, want %v", saga.TransactionId, transactionId)
	}
	
	if saga.SagaType != sagaType {
		t.Errorf("Builder set SagaType = %v, want %v", saga.SagaType, sagaType)
	}
	
	if saga.InitiatedBy != initiatedBy {
		t.Errorf("Builder set InitiatedBy = %v, want %v", saga.InitiatedBy, initiatedBy)
	}
	
	if len(saga.Steps) != 2 {
		t.Errorf("Builder added %v steps, want %v", len(saga.Steps), 2)
	}
	
	// Verify the steps
	if saga.Steps[0].StepId != "step1" || saga.Steps[0].Status != Pending {
		t.Errorf("First step has incorrect properties")
	}
	
	if saga.Steps[1].StepId != "step2" || saga.Steps[1].Status != Completed {
		t.Errorf("Second step has incorrect properties")
	}
}

// Test the new state consistency validation functions
func TestSaga_ValidateStateTransition(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() Saga
		stepIndex     int
		newStatus     Status
		expectError   bool
		errorMessage  string
	}{
		{
			name: "Valid transition from Pending to Completed",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Completed,
			expectError:  false,
		},
		{
			name: "Valid transition from Pending to Failed",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Failed,
			expectError:  false,
		},
		{
			name: "Valid transition from Completed to Failed (compensation)",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Completed, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Failed,
			expectError:  false,
		},
		{
			name: "Valid transition from Failed to Pending (after compensation)",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Failed, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Pending,
			expectError:  false,
		},
		{
			name: "Invalid transition from Pending to Pending",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Pending,
			expectError:  true,
			errorMessage: "invalid transition from pending to pending",
		},
		{
			name: "Invalid transition from Completed to Pending",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Completed, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Pending,
			expectError:  true,
			errorMessage: "invalid transition from completed to pending",
		},
		{
			name: "Invalid step index",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    5,
			newStatus:    Completed,
			expectError:  true,
			errorMessage: "invalid step index",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := tt.setup()
			err := saga.ValidateStateTransition(tt.stepIndex, tt.newStatus)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSaga_SetStepStatus(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() Saga
		stepIndex     int
		newStatus     Status
		expectError   bool
	}{
		{
			name: "Valid status update",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Completed,
			expectError:  false,
		},
		{
			name: "Invalid status update",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Completed, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			stepIndex:    0,
			newStatus:    Pending,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := tt.setup()
			originalUpdatedAt := saga.Steps[tt.stepIndex].UpdatedAt
			time.Sleep(1 * time.Millisecond) // Ensure time difference

			err := saga.SetStepStatus(tt.stepIndex, tt.newStatus)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if saga.Steps[tt.stepIndex].Status != tt.newStatus {
					t.Errorf("Status was not updated. Expected %v, got %v", tt.newStatus, saga.Steps[tt.stepIndex].Status)
				}
				if !saga.Steps[tt.stepIndex].UpdatedAt.After(originalUpdatedAt) {
					t.Errorf("UpdatedAt was not updated")
				}
			}
		})
	}
}

func TestSaga_ValidateStateConsistency(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() Saga
		expectError   bool
		errorMessage  string
	}{
		{
			name: "Valid saga state",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Completed, AwardAsset, AwardItemActionPayload{}).
					AddStep("step2", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
			},
			expectError: false,
		},
		{
			name: "Invalid step ordering",
			setup: func() Saga {
				saga := NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					AddStep("step2", Completed, AwardAsset, AwardItemActionPayload{}).
					Build()
				return saga
			},
			expectError:  true,
			errorMessage: "invalid step ordering",
		},
		{
			name: "Duplicate step IDs",
			setup: func() Saga {
				saga := NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					AddStep("step1", Pending, AwardAsset, AwardItemActionPayload{}).
					Build()
				return saga
			},
			expectError:  true,
			errorMessage: "duplicate step ID",
		},
		{
			name: "Failing saga with multiple failed steps",
			setup: func() Saga {
				saga := NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("step1", Failed, AwardAsset, AwardItemActionPayload{}).
					AddStep("step2", Failed, AwardAsset, AwardItemActionPayload{}).
					Build()
				return saga
			},
			expectError:  true,
			errorMessage: "saga is failing but has 2 failed steps, expected exactly 1",
		},
		{
			name: "Empty action",
			setup: func() Saga {
				saga := NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					Build()
				// Manually add a step with empty action
				saga.Steps = []Step[any]{{
					StepId:    "step1",
					Status:    Pending,
					Action:    "",
					Payload:   AwardItemActionPayload{},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}}
				return saga
			},
			expectError:  true,
			errorMessage: "empty action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := tt.setup()
			err := saga.ValidateStateConsistency()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCreateAndEquipAssetAction(t *testing.T) {
	// Test that CreateAndEquipAsset action is defined and has correct string value
	t.Run("Action constant value", func(t *testing.T) {
		expected := "create_and_equip_asset"
		actual := string(CreateAndEquipAsset)
		if actual != expected {
			t.Errorf("CreateAndEquipAsset action = %v, want %v", actual, expected)
		}
	})

	t.Run("Action in action constants", func(t *testing.T) {
		// Test that CreateAndEquipAsset is one of the defined actions
		actions := []Action{
			AwardInventory,
			AwardAsset,
			AwardExperience,
			AwardLevel,
			AwardMesos,
			WarpToRandomPortal,
			WarpToPortal,
			DestroyAsset,
			EquipAsset,
			UnequipAsset,
			ChangeJob,
			CreateSkill,
			UpdateSkill,
			ValidateCharacterState,
			RequestGuildName,
			RequestGuildEmblem,
			RequestGuildDisband,
			RequestGuildCapacityIncrease,
			CreateInvite,
			CreateCharacter,
			CreateAndEquipAsset,
		}
		
		found := false
		for _, action := range actions {
			if action == CreateAndEquipAsset {
				found = true
				break
			}
		}
		
		if !found {
			t.Errorf("CreateAndEquipAsset action not found in actions list")
		}
	})
}

func TestCreateAndEquipAssetPayload(t *testing.T) {
	// Test CreateAndEquipAssetPayload struct construction and validation
	t.Run("Valid payload construction", func(t *testing.T) {
		payload := CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: ItemPayload{
				TemplateId: 1302000,
				Quantity:   1,
			},
		}
		
		// Verify field values
		if payload.CharacterId != 12345 {
			t.Errorf("CharacterId = %v, want %v", payload.CharacterId, 12345)
		}
		
		if payload.Item.TemplateId != 1302000 {
			t.Errorf("Item.TemplateId = %v, want %v", payload.Item.TemplateId, 1302000)
		}
		
		if payload.Item.Quantity != 1 {
			t.Errorf("Item.Quantity = %v, want %v", payload.Item.Quantity, 1)
		}
	})

	t.Run("Zero values", func(t *testing.T) {
		payload := CreateAndEquipAssetPayload{}
		
		if payload.CharacterId != 0 {
			t.Errorf("Zero CharacterId = %v, want %v", payload.CharacterId, 0)
		}
		
		if payload.Item.TemplateId != 0 {
			t.Errorf("Zero Item.TemplateId = %v, want %v", payload.Item.TemplateId, 0)
		}
		
		if payload.Item.Quantity != 0 {
			t.Errorf("Zero Item.Quantity = %v, want %v", payload.Item.Quantity, 0)
		}
	})

	t.Run("Different item types", func(t *testing.T) {
		testCases := []struct {
			name       string
			templateId uint32
			quantity   uint32
		}{
			{"Equipment item", 1302000, 1},
			{"Consumable item", 2000000, 100},
			{"Etc item", 4000000, 50},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payload := CreateAndEquipAssetPayload{
					CharacterId: 99999,
					Item: ItemPayload{
						TemplateId: tc.templateId,
						Quantity:   tc.quantity,
					},
				}
				
				if payload.Item.TemplateId != tc.templateId {
					t.Errorf("Item.TemplateId = %v, want %v", payload.Item.TemplateId, tc.templateId)
				}
				
				if payload.Item.Quantity != tc.quantity {
					t.Errorf("Item.Quantity = %v, want %v", payload.Item.Quantity, tc.quantity)
				}
			})
		}
	})
}

func TestCreateAndEquipAssetStepSerialization(t *testing.T) {
	// Test JSON marshaling and unmarshaling of CreateAndEquipAsset steps
	t.Run("JSON marshaling", func(t *testing.T) {
		payload := CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: ItemPayload{
				TemplateId: 1302000,
				Quantity:   1,
			},
		}
		
		step := Step[CreateAndEquipAssetPayload]{
			StepId:    "create_and_equip_1",
			Status:    Pending,
			Action:    CreateAndEquipAsset,
			Payload:   payload,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		// Marshal to JSON
		jsonData, err := json.Marshal(step)
		if err != nil {
			t.Errorf("Failed to marshal step to JSON: %v", err)
		}
		
		// Verify JSON contains expected fields
		jsonStr := string(jsonData)
		if !strings.Contains(jsonStr, "create_and_equip_asset") {
			t.Errorf("JSON does not contain action type: %s", jsonStr)
		}
		
		if !strings.Contains(jsonStr, "12345") {
			t.Errorf("JSON does not contain characterId: %s", jsonStr)
		}
		
		if !strings.Contains(jsonStr, "1302000") {
			t.Errorf("JSON does not contain templateId: %s", jsonStr)
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		jsonData := `{
			"stepId": "create_and_equip_1",
			"status": "pending",
			"action": "create_and_equip_asset",
			"payload": {
				"characterId": 12345,
				"item": {
					"templateId": 1302000,
					"quantity": 1
				}
			},
			"createdAt": "2023-01-01T00:00:00Z",
			"updatedAt": "2023-01-01T00:00:00Z"
		}`
		
		var step Step[any]
		err := json.Unmarshal([]byte(jsonData), &step)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON to step: %v", err)
		}
		
		// Verify step fields
		if step.StepId != "create_and_equip_1" {
			t.Errorf("StepId = %v, want %v", step.StepId, "create_and_equip_1")
		}
		
		if step.Status != Pending {
			t.Errorf("Status = %v, want %v", step.Status, Pending)
		}
		
		if step.Action != CreateAndEquipAsset {
			t.Errorf("Action = %v, want %v", step.Action, CreateAndEquipAsset)
		}
		
		// Verify payload by type assertion
		payload, ok := step.Payload.(CreateAndEquipAssetPayload)
		if !ok {
			t.Errorf("Payload is not CreateAndEquipAssetPayload type")
		}
		
		if payload.CharacterId != 12345 {
			t.Errorf("Payload.CharacterId = %v, want %v", payload.CharacterId, 12345)
		}
		
		if payload.Item.TemplateId != 1302000 {
			t.Errorf("Payload.Item.TemplateId = %v, want %v", payload.Item.TemplateId, 1302000)
		}
		
		if payload.Item.Quantity != 1 {
			t.Errorf("Payload.Item.Quantity = %v, want %v", payload.Item.Quantity, 1)
		}
	})
}

func TestCreateAndEquipAssetStepBuilder(t *testing.T) {
	// Test using builder pattern with CreateAndEquipAsset steps
	t.Run("Builder with CreateAndEquipAsset step", func(t *testing.T) {
		transactionId := uuid.New()
		payload := CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: ItemPayload{
				TemplateId: 1302000,
				Quantity:   1,
			},
		}
		
		saga := NewBuilder().
			SetTransactionId(transactionId).
			SetSagaType(InventoryTransaction).
			SetInitiatedBy("test").
			AddStep("create_and_equip_1", Pending, CreateAndEquipAsset, payload).
			Build()
		
		// Verify saga properties
		if saga.TransactionId != transactionId {
			t.Errorf("TransactionId = %v, want %v", saga.TransactionId, transactionId)
		}
		
		if saga.SagaType != InventoryTransaction {
			t.Errorf("SagaType = %v, want %v", saga.SagaType, InventoryTransaction)
		}
		
		if len(saga.Steps) != 1 {
			t.Errorf("Steps length = %v, want %v", len(saga.Steps), 1)
		}
		
		step := saga.Steps[0]
		if step.StepId != "create_and_equip_1" {
			t.Errorf("Step.StepId = %v, want %v", step.StepId, "create_and_equip_1")
		}
		
		if step.Action != CreateAndEquipAsset {
			t.Errorf("Step.Action = %v, want %v", step.Action, CreateAndEquipAsset)
		}
		
		if step.Status != Pending {
			t.Errorf("Step.Status = %v, want %v", step.Status, Pending)
		}
		
		// Verify payload
		stepPayload, ok := step.Payload.(CreateAndEquipAssetPayload)
		if !ok {
			t.Errorf("Step.Payload is not CreateAndEquipAssetPayload type")
		}
		
		if stepPayload.CharacterId != 12345 {
			t.Errorf("Step.Payload.CharacterId = %v, want %v", stepPayload.CharacterId, 12345)
		}
	})
}

func TestCreateAndEquipAssetPayloadValidation(t *testing.T) {
	// Test edge cases and validation scenarios for CreateAndEquipAssetPayload
	t.Run("Edge case values", func(t *testing.T) {
		testCases := []struct {
			name        string
			characterId uint32
			templateId  uint32
			quantity    uint32
			description string
		}{
			{"Max character ID", 4294967295, 1302000, 1, "Maximum uint32 character ID"},
			{"Min character ID", 1, 1302000, 1, "Minimum character ID"},
			{"Max template ID", 4294967295, 1302000, 1, "Maximum uint32 template ID"},
			{"Min template ID", 1, 1302000, 1, "Minimum template ID"},
			{"Max quantity", 4294967295, 1302000, 1, "Maximum uint32 quantity"},
			{"Min quantity", 1, 1302000, 1, "Minimum quantity"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payload := CreateAndEquipAssetPayload{
					CharacterId: tc.characterId,
					Item: ItemPayload{
						TemplateId: tc.templateId,
						Quantity:   tc.quantity,
					},
				}
				
				if payload.CharacterId != tc.characterId {
					t.Errorf("CharacterId = %v, want %v for %s", payload.CharacterId, tc.characterId, tc.description)
				}
				
				if payload.Item.TemplateId != tc.templateId {
					t.Errorf("Item.TemplateId = %v, want %v for %s", payload.Item.TemplateId, tc.templateId, tc.description)
				}
				
				if payload.Item.Quantity != tc.quantity {
					t.Errorf("Item.Quantity = %v, want %v for %s", payload.Item.Quantity, tc.quantity, tc.description)
				}
			})
		}
	})
}

func TestSaga_CreateAndEquipAssetStateConsistency(t *testing.T) {
	// Test specifically for CreateAndEquipAsset compound operation state consistency
	tests := []struct {
		name          string
		setup         func() Saga
		expectedValid bool
		description   string
	}{
		{
			name: "Valid CreateAndEquipAsset with auto-generated step",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("create_and_equip_1", Completed, CreateAndEquipAsset, CreateAndEquipAssetPayload{
						CharacterId: 12345,
						Item: ItemPayload{TemplateId: 1001, Quantity: 1},
					}).
					AddStep("auto_equip_step_1234567890", Pending, EquipAsset, EquipAssetPayload{
						CharacterId: 12345,
						InventoryType: 1,
						Source: 5,
						Destination: -1,
					}).
					Build()
			},
			expectedValid: true,
			description:   "Completed CreateAndEquipAsset followed by pending auto-equip step",
		},
		{
			name: "Failed CreateAndEquipAsset without auto-generated step",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("create_and_equip_1", Failed, CreateAndEquipAsset, CreateAndEquipAssetPayload{
						CharacterId: 12345,
						Item: ItemPayload{TemplateId: 1001, Quantity: 1},
					}).
					Build()
			},
			expectedValid: true,
			description:   "Failed CreateAndEquipAsset without auto-equip step (asset creation failed)",
		},
		{
			name: "CreateAndEquipAsset with failed auto-equip step",
			setup: func() Saga {
				return NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(InventoryTransaction).
					SetInitiatedBy("test").
					AddStep("create_and_equip_1", Completed, CreateAndEquipAsset, CreateAndEquipAssetPayload{
						CharacterId: 12345,
						Item: ItemPayload{TemplateId: 1001, Quantity: 1},
					}).
					AddStep("auto_equip_step_1234567890", Failed, EquipAsset, EquipAssetPayload{
						CharacterId: 12345,
						InventoryType: 1,
						Source: 5,
						Destination: -1,
					}).
					Build()
			},
			expectedValid: true,
			description:   "CreateAndEquipAsset with failed auto-equip step (equipment failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saga := tt.setup()
			err := saga.ValidateStateConsistency()

			if tt.expectedValid {
				if err != nil {
					t.Errorf("Expected valid state for %s, but got error: %v", tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected invalid state for %s, but validation passed", tt.description)
				}
			}
		})
	}
}