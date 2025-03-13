package inventory

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

	ChangedTypeAdd     = "INVENTORY_CHANGED_TYPE_ADD"
	ChangedTypeUpdate  = "INVENTORY_CHANGED_TYPE_UPDATE"
	ChangedTypeRemove  = "INVENTORY_CHANGED_TYPE_REMOVE"
	ChangedTypeMove    = "INVENTORY_CHANGED_TYPE_MOVE"
	ChangedTypeReserve = "INVENTORY_CHANGED_TYPE_RESERVE"
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

type inventoryChangedItemUpdateBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

type inventoryChangedItemMoveBody struct {
	ItemId  uint32 `json:"itemId"`
	OldSlot int16  `json:"oldSlot"`
}

type inventoryChangedItemRemoveBody struct {
	ItemId uint32 `json:"itemId"`
}

type inventoryChangedItemReserveBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}
