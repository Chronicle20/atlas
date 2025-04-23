package compartment

import "github.com/google/uuid"

const (
	EnvCommandTopic          = "COMMAND_TOPIC_COMPARTMENT"
	CommandRequestReserve    = "REQUEST_RESERVE"
	CommandConsume           = "CONSUME"
	CommandDestroy           = "DESTROY"
	CommandCancelReservation = "CANCEL_RESERVATION"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type RequestReserveCommandBody struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	Items         []ItemBody `json:"items"`
}

type ItemBody struct {
	Source   int16  `json:"source"`
	ItemId   uint32 `json:"itemId"`
	Quantity int16  `json:"quantity"`
}

type ConsumeCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}

type DestroyCommandBody struct {
	Slot int16 `json:"slot"`
}

type CancelReservationCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Slot          int16     `json:"slot"`
}

const (
	EnvEventTopicStatus                 = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeReserved             = "RESERVED"
	StatusEventTypeReservationCancelled = "RESERVATION_CANCELLED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ReservedEventBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ItemId        uint32    `json:"itemId"`
	Slot          int16     `json:"slot"`
	Quantity      uint32    `json:"quantity"`
}
type ReservationCancelledEventBody struct {
	ItemId   uint32 `json:"itemId"`
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}
