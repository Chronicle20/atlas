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
type Saga[T any] struct {
	TransactionID uuid.UUID `json:"transaction_id"` // Unique ID for the transaction
	SagaType      Type      `json:"saga_type"`      // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string    `json:"initiated_by"`   // Who initiated the saga (e.g., NPC ID, user)
	Steps         []Step[T] `json:"steps"`          // List of steps in the saga
}

type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Step represents a single step within a saga.
type Step[T any] struct {
	StepID    string    `json:"step_id"`    // Unique ID for the step
	Status    Status    `json:"status"`     // Status of the step (e.g., pending, completed, failed)
	Action    Action    `json:"action"`     // The Action to be taken (e.g., validate_inventory, deduct_inventory)
	Payload   T         `json:"payload"`    // Data required for the action (specific to the action type)
	CreatedAt time.Time `json:"created_at"` // Timestamp of when the step was created
	UpdatedAt time.Time `json:"updated_at"` // Timestamp of the last update to the step
}

// AwardItemActionPayload represents the data needed to execute a specific action in a step.
type AwardItemActionPayload struct {
	CharacterId uint32        `json:"character_id"` // Character ID associated with the action
	Items       []ItemPayload `json:"items"`        // List of items involved in the action
}

// ItemPayload represents an individual item in a transaction, such as in inventory manipulation.
type ItemPayload struct {
	TemplateID uint32 `json:"template_id"` // Template ID of the item
	Quantity   int    `json:"quantity"`    // Quantity of the item
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
