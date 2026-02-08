package compartment

import (
	"atlas-storage/kafka/message"

	"github.com/Chronicle20/atlas-constants/asset"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_STORAGE_COMPARTMENT"
	CommandAccept   = "ACCEPT"
	CommandRelease  = "RELEASE"
)

// Command represents a storage compartment command (ACCEPT/RELEASE)
type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	AccountId   uint32   `json:"accountId"`
	CharacterId uint32   `json:"characterId,omitempty"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

// AcceptCommandBody contains the data for an ACCEPT command
type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	TemplateId    uint32    `json:"templateId"`
	message.AssetData
}

// ReleaseCommandBody contains the data for a RELEASE command
type ReleaseCommandBody struct {
	TransactionId uuid.UUID      `json:"transactionId"`
	AssetId       asset.Id       `json:"assetId"`
	Quantity      asset.Quantity `json:"quantity"`
}

const (
	EnvEventTopicStatus     = "EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"
)

// StatusEvent represents a storage compartment status event
type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	AccountId   uint32   `json:"accountId"`
	CharacterId uint32   `json:"characterId,omitempty"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

// StatusEventAcceptedBody contains the data for an ACCEPTED event
type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       asset.Id  `json:"assetId"`
	Slot          int16     `json:"slot"`
	InventoryType byte      `json:"inventoryType"`
}

// StatusEventReleasedBody contains the data for a RELEASED event
type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       asset.Id  `json:"assetId"`
	InventoryType byte      `json:"inventoryType"`
}

// StatusEventErrorBody contains the data for an ERROR event
type StatusEventErrorBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ErrorCode     string    `json:"errorCode"`
	Message       string    `json:"message,omitempty"`
}
