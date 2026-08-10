// Package compartment carries the COMMAND_TOPIC_COMPARTMENT envelopes this
// service produces to reserve and release staged assets. Mirrors
// services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go;
// struct names, field names and json tags must match that file exactly. Only
// the commands this service issues are carried over, the same trimming
// atlas-saga-orchestrator's mirror applies.
package compartment

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_COMPARTMENT"
	CommandRequestReserve    = "REQUEST_RESERVE"
	CommandCancelReservation = "CANCEL_RESERVATION"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestReserveCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	// ExpirySeconds is the reservation TTL. Zero means the historical 30s
	// default, so pre-task-205 producers keep working unchanged.
	ExpirySeconds uint32     `json:"expirySeconds"`
	Items         []ItemBody `json:"items"`
}

type ItemBody struct {
	Source   int16  `json:"source"`
	ItemId   uint32 `json:"itemId"`
	Quantity int16  `json:"quantity"`
}

type CancelReservationCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}
