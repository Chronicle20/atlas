package buff

const (
	EnvCommandTopic      = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeCancelAll = "CANCEL_ALL"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CancelAllBody struct {
}
