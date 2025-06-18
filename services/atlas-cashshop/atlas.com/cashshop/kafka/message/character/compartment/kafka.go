package compartment

const (
	EnvCommandTopic         = "COMMAND_TOPIC_COMPARTMENT"
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
