package guild

const (
	EnvCommandTopic                    = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName             = "REQUEST_NAME"
	CommandTypeRequestEmblem           = "REQUEST_EMBLEM"
	CommandTypeRequestDisband          = "REQUEST_DISBAND"
	CommandTypeRequestCapacityIncrease = "REQUEST_CAPACITY_INCREASE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestNameBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type RequestEmblemBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type RequestDisbandBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type RequestCapacityIncreaseBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}
