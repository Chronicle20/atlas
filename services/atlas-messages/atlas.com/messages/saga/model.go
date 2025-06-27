package saga

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// Type the type of saga
type Type string

// Constants for different saga types
const (
	InventoryTransaction Type = "inventory_transaction"
	QuestReward          Type = "quest_reward"
	TradeTransaction     Type = "trade_transaction"
)

// Define a custom type for Action
type Action string

// Constants for different actions
const (
	AwardInventory Action = "award_inventory"
)

// Saga represents the entire saga transaction.
type Saga struct {
	TransactionId uuid.UUID   `json:"transactionId"` // Unique ID for the transaction
	SagaType      Type        `json:"sagaType"`      // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string      `json:"initiatedBy"`   // Who initiated the saga (e.g., NPC ID, user)
	Steps         []Step[any] `json:"steps"`         // List of steps in the saga
}

func (s *Saga) Failing() bool {
	for _, step := range s.Steps {
		if step.Status == Failed {
			return true
		}
	}
	return false
}

func (s *Saga) GetCurrentStep() (Step[any], bool) {
	for idx, step := range s.Steps {
		if step.Status == Pending {
			return s.Steps[idx], true
		}
	}
	return Step[any]{}, false
}

// FindFurthestCompletedStepIndex returns the index of the furthest completed step (last one with status "completed")
// Returns -1 if no completed step is found
func (s *Saga) FindFurthestCompletedStepIndex() int {
	furthestCompletedIndex := -1
	for i := len(s.Steps) - 1; i >= 0; i-- {
		if s.Steps[i].Status == Completed {
			furthestCompletedIndex = i
			break
		}
	}
	return furthestCompletedIndex
}

// FindEarliestPendingStepIndex returns the index of the earliest pending step (first one with status "pending")
// Returns -1 if no pending step is found
func (s *Saga) FindEarliestPendingStepIndex() int {
	earliestPendingIndex := -1
	for i := 0; i < len(s.Steps); i++ {
		if s.Steps[i].Status == Pending {
			earliestPendingIndex = i
			break
		}
	}
	return earliestPendingIndex
}

// SetStepStatus sets the status of a step at the given index
func (s *Saga) SetStepStatus(index int, status Status) {
	if index >= 0 && index < len(s.Steps) {
		s.Steps[index].Status = status
	}
}

type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Step represents a single step within a saga.
type Step[T any] struct {
	StepId    string    `json:"stepId"`    // Unique ID for the step
	Status    Status    `json:"status"`    // Status of the step (e.g., pending, completed, failed)
	Action    Action    `json:"action"`    // The Action to be taken (e.g., validate_inventory, deduct_inventory)
	Payload   T         `json:"payload"`   // Data required for the action (specific to the action type)
	CreatedAt time.Time `json:"createdAt"` // Timestamp of when the step was created
	UpdatedAt time.Time `json:"updatedAt"` // Timestamp of the last update to the step
}

// AwardItemActionPayload represents the data needed to execute a specific action in a step.
type AwardItemActionPayload struct {
	CharacterId uint32      `json:"characterId"` // Character ID associated with the action
	Item        ItemPayload `json:"item"`        // List of items involved in the action
}

// ItemPayload represents an individual item in a transaction, such as in inventory manipulation.
type ItemPayload struct {
	TemplateId uint32 `json:"templateId"` // Template ID of the item
	Quantity   uint32 `json:"quantity"`   // Quantity of the item
}

// Custom UnmarshalJSON for Step[T] to handle the generics
func (s *Step[T]) UnmarshalJSON(data []byte) error {
	type Alias Step[T] // Alias to avoid recursion
	aux := &struct {
		Payload json.RawMessage `json:"payload"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	// Unmarshal the generic part (excluding Payload first)
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Now handle the Payload field based on the Action type (you can customize this)
	switch s.Action {
	case AwardInventory:
		var payload AwardItemActionPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	default:
		return fmt.Errorf("unknown action: %s", s.Action)
	}

	return nil
}
