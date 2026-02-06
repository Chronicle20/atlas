package compartment

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CASH_COMPARTMENT"
	CommandAccept   = "ACCEPT"
	CommandRelease  = "RELEASE"
)

type Command[E any] struct {
	AccountId       uint32 `json:"accountId"`
	CharacterId     uint32 `json:"characterId"`
	CompartmentType byte   `json:"compartmentType"`
	Type            string `json:"type"`
	Body            E      `json:"body"`
}

type AcceptCommandBody struct {
	TransactionId uuid.UUID       `json:"transactionId"`
	CompartmentId uuid.UUID       `json:"compartmentId"`
	CashId        int64           `json:"cashId"` // Preserved CashId from source item
	TemplateId    uint32          `json:"templateId"`
	ReferenceId   uint32          `json:"referenceId"`
	ReferenceType string          `json:"referenceType"`
	ReferenceData json.RawMessage `json:"referenceData,omitempty"`
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	CashId        int64     `json:"cashId"`     // CashId for client notification correlation
	TemplateId    uint32    `json:"templateId"` // Item template ID for client notification
}

const (
	EnvEventTopicStatus     = "EVENT_TOPIC_CASH_COMPARTMENT_STATUS"
	StatusEventTypeCreated  = "CREATED"
	StatusEventTypeUpdated  = "UPDATED"
	StatusEventTypeDeleted  = "DELETED"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"
)

// StatusEvent represents a cash compartment status event
// Contains accountId and characterId to allow channel service to find the session and notify the client
type StatusEvent[E any] struct {
	AccountId       uint32    `json:"accountId"`
	CharacterId     uint32    `json:"characterId"`
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	Type            string    `json:"type"`
	Body            E         `json:"body"`
}

// StatusEventCreatedBody contains information for compartment creation events
// According to the requirements, it should include the capacity
type StatusEventCreatedBody struct {
	Capacity uint32 `json:"capacity"`
}

// StatusEventUpdatedBody contains information for compartment update events
type StatusEventUpdatedBody struct {
	Capacity uint32 `json:"capacity"`
}

// StatusEventDeletedBody is an empty body for compartment deletion events
type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}

type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uuid.UUID `json:"assetId"`
}

type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	CashId        int64     `json:"cashId"`     // CashId for client notification correlation
	TemplateId    uint32    `json:"templateId"` // Item template ID for client notification
}

type StatusEventErrorBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}
