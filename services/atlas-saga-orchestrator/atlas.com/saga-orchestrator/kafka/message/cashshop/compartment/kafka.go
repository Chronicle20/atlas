package compartment

import (
	"encoding/json"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_CASH_COMPARTMENT"
	EnvEventTopicStatus     = "EVENT_TOPIC_CASH_COMPARTMENT_STATUS"
	CommandAccept           = "ACCEPT"
	CommandRelease          = "RELEASE"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"
)

// Command represents a cash shop compartment command (ACCEPT/RELEASE)
type Command[E any] struct {
	AccountId       uint32 `json:"accountId"`
	CharacterId     uint32 `json:"characterId"`
	CompartmentType byte   `json:"compartmentType"`
	Type            string `json:"type"`
	Body            E      `json:"body"`
}

// AcceptCommandBody contains the data for an ACCEPT command
type AcceptCommandBody struct {
	TransactionId uuid.UUID       `json:"transactionId"`
	CompartmentId uuid.UUID       `json:"compartmentId"`
	CashId        int64           `json:"cashId"`                  // Preserved CashId from source item
	TemplateId    uint32          `json:"templateId"`
	ReferenceId   uint32          `json:"referenceId"`
	ReferenceType string          `json:"referenceType"`
	ReferenceData json.RawMessage `json:"referenceData,omitempty"`
}

// ReleaseCommandBody contains the data for a RELEASE command
type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	CashId        int64     `json:"cashId"`     // CashId for client notification correlation
	TemplateId    uint32    `json:"templateId"` // Item template ID for client notification
}

// StatusEvent represents a cash compartment status event
type StatusEvent[E any] struct {
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	Type            string    `json:"type"`
	Body            E         `json:"body"`
}

// StatusEventAcceptedBody contains the data for an ACCEPTED event
type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

// StatusEventReleasedBody contains the data for a RELEASED event
type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

// StatusEventErrorBody contains the data for an ERROR event
type StatusEventErrorBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}
