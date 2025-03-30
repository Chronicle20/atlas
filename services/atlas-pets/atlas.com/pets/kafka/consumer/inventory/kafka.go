package inventory

const (
	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd    = "ADDED"
	ChangedTypeRemove = "REMOVED"
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

type inventoryChangedItemRemoveBody struct {
	ItemId uint32 `json:"itemId"`
}
