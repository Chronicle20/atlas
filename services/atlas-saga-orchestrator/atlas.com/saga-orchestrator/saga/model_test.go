package saga

import (
	"github.com/google/uuid"
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