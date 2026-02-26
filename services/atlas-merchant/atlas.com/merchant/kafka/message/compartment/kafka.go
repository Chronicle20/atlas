package compartment

import (
	asset2 "atlas-merchant/kafka/message/asset"

	"github.com/google/uuid"
)

const (
	EnvCommandTopic     = "COMMAND_TOPIC_COMPARTMENT"
	EnvEventTopicStatus = "EVENT_TOPIC_COMPARTMENT_STATUS"

	CommandAccept  = "ACCEPT"
	CommandRelease = "RELEASE"

	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"

	AcceptCommandFailed  = "ACCEPT_COMMAND_FAILED"
	ReleaseCommandFailed = "RELEASE_COMMAND_FAILED"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	TemplateId    uint32    `json:"templateId"`
	asset2.AssetData
}

type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	Quantity      uint32    `json:"quantity"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
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
