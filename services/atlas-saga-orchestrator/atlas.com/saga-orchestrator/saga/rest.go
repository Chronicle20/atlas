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

// PayloadUnmarshaler is a function type for unmarshaling payloads
type PayloadUnmarshaler func(interface{}) (any, error)

// payloadUnmarshalers maps action types to their payload unmarshalers
var payloadUnmarshalers = map[Action]PayloadUnmarshaler{
	AwardInventory:     unmarshalAwardInventoryPayload,
	AwardExperience:    unmarshalAwardExperiencePayload,
	AwardLevel:         unmarshalAwardLevelPayload,
	AwardMesos:         unmarshalAwardMesosPayload,
	WarpToRandomPortal: unmarshalWarpToRandomPortalPayload,
	WarpToPortal:       unmarshalWarpToPortalPayload,
	DestroyAsset:       unmarshalDestroyAssetPayload,
}

// parseTime parses a time string in RFC3339 format, returning current time if parsing fails
func parseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}

// unmarshalPayload unmarshals a payload based on the action type
func unmarshalPayload(action Action, rawPayload interface{}) (any, error) {
	// If no unmarshaler is registered for this action, return the payload as is
	unmarshaler, exists := payloadUnmarshalers[action]
	if !exists {
		return rawPayload, nil
	}

	return unmarshaler(rawPayload)
}

// unmarshalGenericPayload is a helper function for unmarshaling payloads
func unmarshalGenericPayload[T any](rawPayload interface{}) (T, error) {
	var result T

	// Marshal the raw payload to JSON
	pbs, err := json.Marshal(rawPayload)
	if err != nil {
		return result, err
	}

	// Unmarshal the JSON to the specific type
	err = json.Unmarshal(pbs, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// Unmarshalers for specific payload types
func unmarshalAwardInventoryPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[AwardItemActionPayload](rawPayload)
}

func unmarshalAwardExperiencePayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[AwardExperiencePayload](rawPayload)
}

func unmarshalAwardLevelPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[AwardLevelPayload](rawPayload)
}

func unmarshalWarpToRandomPortalPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[WarpToRandomPortalPayload](rawPayload)
}

func unmarshalWarpToPortalPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[WarpToPortalPayload](rawPayload)
}

func unmarshalAwardMesosPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[AwardMesosPayload](rawPayload)
}

func unmarshalDestroyAssetPayload(rawPayload interface{}) (any, error) {
	return unmarshalGenericPayload[DestroyAssetPayload](rawPayload)
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (Saga, error) {
	steps := make([]Step[any], len(r.Steps))
	for i, step := range r.Steps {
		// Parse timestamps
		createdAt := parseTime(step.CreatedAt)
		updatedAt := parseTime(step.UpdatedAt)

		// Unmarshal payload based on action type
		payload, err := unmarshalPayload(step.Action, step.Payload)
		if err != nil {
			return Saga{}, err
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
