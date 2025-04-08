package compartment

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Type     byte   `json:"type"`
	Capacity uint32 `json:"capacity"`
}

type DeletedStatusEventBody struct {
}

const (
	EnvCommandTopic          = "COMMAND_TOPIC_COMPARTMENT"
	CommandEquip             = "EQUIP"
	CommandUnequip           = "UNEQUIP"
	CommandMove              = "MOVE"
	CommandDrop              = "DROP"
	CommandRequestReserve    = "REQUEST_RESERVE"
	CommandConsume           = "CONSUME"
	CommandDestroy           = "DESTROY"
	CommandCancelReservation = "CANCEL_RESERVATION"
	CommandIncreaseCapacity  = "INCREASE_CAPACITY"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type EquipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type UnequipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type MoveCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type DropCommandBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Source    int16  `json:"source"`
	Quantity  int16  `json:"quantity"`
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

type IncreaseCapacityCommandBody struct {
	Amount uint32 `json:"amount"`
}
