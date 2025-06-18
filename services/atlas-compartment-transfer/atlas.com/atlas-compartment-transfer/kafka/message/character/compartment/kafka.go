package compartment

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_COMPARTMENT"
	CommandAccept   = "ACCEPT"
	CommandRelease  = "RELEASE"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ReferenceId   uint32    `json:"referenceId"`
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
}

const (
	EnvEventTopicStatus     = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"

	AcceptCommandFailed  = "ACCEPT_COMMAND_FAILED"
	ReleaseCommandFailed = "RELEASE_COMMAND_FAILED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AcceptedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

type ReleasedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

type ErrorEventBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}
