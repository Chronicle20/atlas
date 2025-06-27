package saga

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

// RestModel is the JSON:API resource for sagas
type RestModel struct {
	TransactionID uuid.UUID       `json:"transactionId"` // Unique ID for the transaction
	SagaType      Type            `json:"sagaType"`      // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string          `json:"initiatedBy"`   // Who initiated the saga (e.g., NPC ID, user)
	Steps         []StepRestModel `json:"steps"`         // List of steps in the saga
}

// StepRestModel is the JSON:API resource for saga steps
type StepRestModel struct {
	StepID    string      `json:"stepId"`    // Unique ID for the step
	Status    Status      `json:"status"`    // Status of the step (e.g., pending, completed, failed)
	Action    Action      `json:"action"`    // The Action to be taken (e.g., validate_inventory, deduct_inventory)
	Payload   interface{} `json:"payload"`   // Data required for the action (specific to the action type)
	CreatedAt string      `json:"createdAt"` // Timestamp of when the step was created
	UpdatedAt string      `json:"updatedAt"` // Timestamp of the last update to the step
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.TransactionID.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	r.TransactionID = transactionID
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "sagas"
}

// Transform converts a domain model to a REST model
func Transform(s Saga) (RestModel, error) {
	steps := make([]StepRestModel, len(s.Steps))
	for i, step := range s.Steps {
		steps[i] = StepRestModel{
			StepID:    step.StepId,
			Status:    step.Status,
			Action:    step.Action,
			Payload:   step.Payload,
			CreatedAt: step.CreatedAt.Format(time.RFC3339),
			UpdatedAt: step.UpdatedAt.Format(time.RFC3339),
		}
	}

	return RestModel{
		TransactionID: s.TransactionId,
		SagaType:      s.SagaType,
		InitiatedBy:   s.InitiatedBy,
		Steps:         steps,
	}, nil
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (Saga, error) {
	steps := make([]Step[any], len(r.Steps))
	for i, step := range r.Steps {
		createdAt, err := time.Parse(time.RFC3339, step.CreatedAt)
		if err != nil {
			createdAt = time.Now()
		}

		updatedAt, err := time.Parse(time.RFC3339, step.UpdatedAt)
		if err != nil {
			updatedAt = time.Now()
		}

		var payload any
		if step.Action == AwardInventory {
			var pbs []byte
			pbs, err = json.Marshal(step.Payload)
			if err != nil {
				return Saga{}, err
			}
			var payloadAwardItemActionPayload AwardItemActionPayload
			err = json.Unmarshal(pbs, &payloadAwardItemActionPayload)
			if err != nil {
				return Saga{}, err
			}
			payload = payloadAwardItemActionPayload
		} else if step.Action == WarpToRandomPortal {
			var pbs []byte
			pbs, err = json.Marshal(step.Payload)
			if err != nil {
				return Saga{}, err
			}
			var payloadWarpToRandomPortalPayload WarpToRandomPortalPayload
			err = json.Unmarshal(pbs, &payloadWarpToRandomPortalPayload)
			if err != nil {
				return Saga{}, err
			}
			payload = payloadWarpToRandomPortalPayload
		} else if step.Action == WarpToPortal {
			var pbs []byte
			pbs, err = json.Marshal(step.Payload)
			if err != nil {
				return Saga{}, err
			}
			var payloadWarpToPortalPayload WarpToPortalPayload
			err = json.Unmarshal(pbs, &payloadWarpToPortalPayload)
			if err != nil {
				return Saga{}, err
			}
			payload = payloadWarpToPortalPayload
		} else {
			payload = step.Payload
		}

		steps[i] = Step[any]{
			StepId:    step.StepID,
			Status:    step.Status,
			Action:    step.Action,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Payload:   payload,
		}
	}

	return Saga{
		TransactionId: r.TransactionID,
		SagaType:      r.SagaType,
		InitiatedBy:   r.InitiatedBy,
		Steps:         steps,
	}, nil
}
