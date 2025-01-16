package list

const (
	EnvCommandTopic   = "COMMAND_TOPIC_BUDDY_LIST"
	CommandTypeCreate = "CREATE"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type createCommandBody struct {
	Capacity uint32 `json:"capacity"`
}
