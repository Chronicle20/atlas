package buddylist

const (
	// EnvCommandTopic defines the environment variable for the buddy list command topic
	EnvCommandTopic             = "COMMAND_TOPIC_BUDDY_LIST"
	// CommandTypeIncreaseCapacity is the command type for increasing buddy list capacity
	CommandTypeIncreaseCapacity = "INCREASE_CAPACITY"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// IncreaseCapacityCommandBody represents the body of an increase capacity command.
// This command is used to increase a character's buddy list capacity.
type IncreaseCapacityCommandBody struct {
	// NewCapacity is the new capacity value that must be greater than the current capacity
	NewCapacity byte `json:"newCapacity"`
}
