package guild

const (
	EnvCommandTopic                    = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName             = "REQUEST_NAME"
	CommandTypeRequestEmblem           = "REQUEST_EMBLEM"
	CommandTypeRequestDisband          = "REQUEST_DISBAND"
	CommandTypeRequestCapacityIncrease = "REQUEST_CAPACITY_INCREASE"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestNameBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type requestEmblemBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type requestDisbandBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type requestCapacityIncreaseBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}
