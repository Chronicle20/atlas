package inventory

import "github.com/google/uuid"

// Mirrors services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go
// so the orchestrator can deserialise events produced by atlas-inventory.

const (
	EnvEventTopicInventoryStatus  = "EVENT_TOPIC_INVENTORY_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"
	StatusEventTypeDeleted        = "DELETED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}

type DeletedStatusEventBody struct {
}
