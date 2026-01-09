package compartment

import (
	"encoding/json"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_STORAGE_COMPARTMENT"
	EnvEventTopicStatus     = "EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS"
	CommandAccept           = "ACCEPT"
	CommandRelease          = "RELEASE"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"
)

// Command represents a storage compartment command (ACCEPT/RELEASE)
type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// AcceptCommandBody contains the data for an ACCEPT command
type AcceptCommandBody struct {
	TransactionId uuid.UUID       `json:"transactionId"`
	Slot          int16           `json:"slot"`
	TemplateId    uint32          `json:"templateId"`
	ReferenceId   uint32          `json:"referenceId"`    // For equipables/pets - points to external service data
	ReferenceType string          `json:"referenceType"`  // "EQUIPABLE", "CONSUMABLE", "SETUP", "ETC", "CASH", "PET"
	ReferenceData json.RawMessage `json:"referenceData,omitempty"` // Type-specific data based on ReferenceType
}

// ReleaseCommandBody contains the data for a RELEASE command
type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
}

// StatusEvent represents a storage compartment status event
type StatusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// StatusEventAcceptedBody contains the data for an ACCEPTED event
type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	Slot          int16     `json:"slot"`
}

// StatusEventReleasedBody contains the data for a RELEASED event
type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
}

// StatusEventErrorBody contains the data for an ERROR event
type StatusEventErrorBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ErrorCode     string    `json:"errorCode"`
	Message       string    `json:"message,omitempty"`
}
