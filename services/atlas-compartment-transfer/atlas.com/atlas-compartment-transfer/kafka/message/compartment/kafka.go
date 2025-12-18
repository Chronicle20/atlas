package compartment

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopicCompartmentTransfer = "COMMAND_TOPIC_COMPARTMENT_TRANSFER"
	InventoryTypeCharacter             = "CHARACTER"
	InventoryTypeCashShop              = "CASH_SHOP"
)

type TransferCommand struct {
	TransactionId       uuid.UUID `json:"transactionId"`
	AccountId           uint32    `json:"accountId"`
	CharacterId         uint32    `json:"characterId"`
	AssetId             uint32    `json:"assetId"`
	FromCompartmentId   uuid.UUID `json:"fromCompartmentId"`
	FromCompartmentType byte      `json:"fromCompartmentType"`
	FromInventoryType   string    `json:"fromInventoryType"`
	ToCompartmentId     uuid.UUID `json:"toCompartmentId"`
	ToCompartmentType   byte      `json:"toCompartmentType"`
	ToInventoryType     string    `json:"toInventoryType"`
	ReferenceId         uint32    `json:"referenceId"`
}

const (
	EnvEventTopicStatus      = "EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS"
	StatusEventTypeCompleted = "COMPLETED"
)

// StatusEvent represents a compartment transfer status event
type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// StatusEventCompletedBody represents the body of a COMPLETED status event
type StatusEventCompletedBody struct {
	TransactionId   uuid.UUID `json:"transactionId"`
	AccountId       uint32    `json:"accountId"`
	AssetId         uint32    `json:"assetId"`
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	InventoryType   string    `json:"inventoryType"`
}
