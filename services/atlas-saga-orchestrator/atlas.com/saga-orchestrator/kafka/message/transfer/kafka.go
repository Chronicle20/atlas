package transfer

import "github.com/google/uuid"

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
