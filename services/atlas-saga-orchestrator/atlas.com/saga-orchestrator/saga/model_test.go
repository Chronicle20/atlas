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