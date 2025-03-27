package inventory

import (
	"github.com/google/uuid"
	"time"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_INVENTORY"
	CommandEquip    = "EQUIP"
	CommandUnequip  = "UNEQUIP"
	CommandMove     = "MOVE"
	CommandDrop     = "DROP"
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

const (
	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd                  = "ADDED"
	ChangedTypeUpdateQuantity       = "QUANTITY_UPDATED"
	ChangedTypeUpdateAttribute      = "ATTRIBUTE_UPDATED"
	ChangedTypeRemove               = "REMOVED"
	ChangedTypeMove                 = "MOVED"
	ChangedTypeReserve              = "RESERVED"
	ChangedTypeReservationCancelled = "RESERVATION_CANCELLED"
)

type inventoryChangedEvent[M any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType int8   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
	Type          string `json:"type"`
	Body          M      `json:"body"`
	Silent        bool   `json:"silent"`
}

type inventoryChangedItemAddBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type inventoryChangedItemQuantityUpdateBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type inventoryChangedItemAttributeUpdateBody struct {
	ItemId         uint32    `json:"itemId"`
	Strength       uint16    `json:"strength"`
	Dexterity      uint16    `json:"dexterity"`
	Intelligence   uint16    `json:"intelligence"`
	Luck           uint16    `json:"luck"`
	HP             uint16    `json:"hp"`
	MP             uint16    `json:"mp"`
	WeaponAttack   uint16    `json:"weaponAttack"`
	MagicAttack    uint16    `json:"magicAttack"`
	WeaponDefense  uint16    `json:"weaponDefense"`
	MagicDefense   uint16    `json:"magicDefense"`
	Accuracy       uint16    `json:"accuracy"`
	Avoidability   uint16    `json:"avoidability"`
	Hands          uint16    `json:"hands"`
	Speed          uint16    `json:"speed"`
	Jump           uint16    `json:"jump"`
	Slots          uint16    `json:"slots"`
	OwnerName      string    `json:"ownerName"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}

type inventoryChangedItemMoveBody struct {
	ItemId  uint32 `json:"itemId"`
	OldSlot int16  `json:"oldSlot"`
}

type inventoryChangedItemRemoveBody struct {
	ItemId uint32 `json:"itemId"`
}

type inventoryChangedItemReserveBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ItemId        uint32    `json:"itemId"`
	Quantity      uint32    `json:"quantity"`
}

type inventoryChangedItemReservationCancelledBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}
