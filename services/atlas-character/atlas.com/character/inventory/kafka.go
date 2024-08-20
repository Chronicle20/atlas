package inventory

import (
	"atlas-character/tenant"
)

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
	EnvCommandTopicMoveItem    = "COMMAND_TOPIC_MOVE_ITEM"
	EnvCommandTopicDropItem    = "COMMAND_TOPIC_DROP_ITEM"
	EnvEventInventoryChanged   = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd    = "INVENTORY_CHANGED_TYPE_ADD"
	ChangedTypeUpdate = "INVENTORY_CHANGED_TYPE_UPDATE"
	ChangedTypeRemove = "INVENTORY_CHANGED_TYPE_REMOVE"
	ChangedTypeMove   = "INVENTORY_CHANGED_TYPE_MOVE"
)

type equipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
	Destination int16        `json:"destination"`
}

type unequipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
	Destination int16        `json:"destination"`
}

type moveItemCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	CharacterId   uint32       `json:"characterId"`
	InventoryType byte         `json:"inventoryType"`
	Source        int16        `json:"source"`
	Destination   int16        `json:"destination"`
}

type dropItemCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	CharacterId   uint32       `json:"characterId"`
	InventoryType byte         `json:"inventoryType"`
	Source        int16        `json:"source"`
	Quantity      int16        `json:"quantity"`
}

type inventoryChangedEvent[M any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Slot        int16        `json:"slot"`
	Type        string       `json:"type"`
	Body        M            `json:"body"`
	Silent      bool         `json:"silent"`
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
