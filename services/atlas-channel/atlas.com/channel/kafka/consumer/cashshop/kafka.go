package cashshop

const (
	EnvEventTopicStatus                       = "EVENT_TOPIC_CASH_SHOP_STATUS"
	StatusEventTypeInventoryCapacityIncreased = "INVENTORY_CAPACITY_INCREASED"
	StatusEventTypeError                      = "ERROR"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type InventoryCapacityIncreasedBody struct {
	InventoryType byte   `json:"inventoryType"`
	Capacity      uint32 `json:"capacity"`
	Amount        uint32 `json:"amount"`
}

type ErrorEventBody struct {
	Error string `json:"error"`
}
