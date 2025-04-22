package inventory

import "time"

const (
	EnvCommandTopic = "COMMAND_TOPIC_INVENTORY"
	CommandEquip    = "EQUIP"
	CommandUnequip  = "UNEQUIP"
	CommandMove     = "MOVE"
	CommandDrop     = "DROP"
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

const (
	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd                  = "ADDED"
	ChangedTypeUpdateQuantity       = "QUANTITY_UPDATED"
	ChangedTypeUpdateAttribute      = "ATTRIBUTE_UPDATED"
	ChangedTypeRemove               = "REMOVED"
	ChangedTypeMove                 = "MOVED"
	ChangedTypeReservationCancelled = "RESERVATION_CANCELLED"
)

type InventoryChangedEvent[M any] struct {
	CharacterId uint32 `json:"characterId"`
	Slot        int16  `json:"slot"`
	Type        string `json:"type"`
	Body        M      `json:"body"`
	Silent      bool   `json:"silent"`
}

type InventoryChangedItemAddBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type InventoryChangedItemQuantityUpdateBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type InventoryChangedItemAttributeUpdateBody struct {
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

type InventoryChangedItemMoveBody struct {
	ItemId  uint32 `json:"itemId"`
	OldSlot int16  `json:"oldSlot"`
}

type InventoryChangedItemRemoveBody struct {
	ItemId uint32 `json:"itemId"`
}

type InventoryChangedItemReservationCancelledBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}
