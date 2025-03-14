package inventory

const (
	EnvEventInventoryChanged = "EVENT_TOPIC_INVENTORY_CHANGED"

	ChangedTypeAdd                  = "ADDED"
	ChangedTypeUpdate               = "UPDATED"
	ChangedTypeRemove               = "REMOVED"
	ChangedTypeMove                 = "MOVED"
	ChangedTypeReservationCancelled = "RESERVATION_CANCELLED"
)

type inventoryChangedEvent[M any] struct {
	CharacterId uint32 `json:"characterId"`
	Slot        int16  `json:"slot"`
	Type        string `json:"type"`
	Body        M      `json:"body"`
	Silent      bool   `json:"silent"`
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

type inventoryChangedItemReservationCancelledBody struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}
