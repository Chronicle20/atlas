package inventory

import "atlas-channel/tenant"

const (
	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd    = "INVENTORY_CHANGED_TYPE_ADD"
	ChangedTypeUpdate = "INVENTORY_CHANGED_TYPE_UPDATE"
	ChangedTypeRemove = "INVENTORY_CHANGED_TYPE_REMOVE"
	ChangedTypeMove   = "INVENTORY_CHANGED_TYPE_MOVE"
)

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
