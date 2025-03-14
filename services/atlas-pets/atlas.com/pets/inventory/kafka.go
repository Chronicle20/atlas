package inventory

import "github.com/google/uuid"

const (
	EnvCommandTopic          = "COMMAND_TOPIC_INVENTORY"
	CommandConsume           = "CONSUME"
	CommandCancelReservation = "CANCEL_RESERVATION"
)

type command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type consumeCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}

type cancelReservationCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}
