package inventory

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVENTORY"
	CommandIncreaseCapacity = "INCREASE_CAPACITY"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type IncreaseCapacityCommandBody struct {
	Amount uint32 `json:"amount"`
}
