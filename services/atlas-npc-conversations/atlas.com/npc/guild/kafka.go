package guild

const (
	EnvCommandTopic        = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName = "REQUEST_NAME"
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
