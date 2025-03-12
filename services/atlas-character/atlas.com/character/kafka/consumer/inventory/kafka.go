package inventory

import "github.com/google/uuid"

const (
	EnvCommandTopic       = "COMMAND_TOPIC_INVENTORY"
	CommandEquip          = "EQUIP"
	CommandUnequip        = "UNEQUIP"
	CommandMove           = "MOVE"
	CommandDrop           = "DROP"
	CommandRequestReserve = "REQUEST_RESERVE"
)

type command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type equipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type unequipCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type moveCommandBody struct {
	Source      int16 `json:"source"`
	Destination int16 `json:"destination"`
}

type dropCommandBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Source    int16  `json:"source"`
	Quantity  int16  `json:"quantity"`
}

type requestReserveCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Source        int16     `json:"source"`
	ItemId        uint32    `json:"itemId"`
	Quantity      int16     `json:"quantity"`
}
