package compartment

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CASH_COMPARTMENT"
	CommandAccept   = "ACCEPT"
	CommandRelease  = "RELEASE"
)

type Command[E any] struct {
	AccountId       uint32 `json:"accountId"`
	CompartmentType byte   `json:"compartmentType"`
	Type            string `json:"type"`
	Body            E      `json:"body"`
}

type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	ReferenceId   uint32    `json:"referenceId"`
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
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
// According to the requirements, it should always contain the compartmentId and type
type StatusEvent[E any] struct {
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
}

type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

type StatusEventErrorBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}
