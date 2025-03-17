package character

const (
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"
	CommandChangeHP = "CHANGE_HP"
	CommandChangeMP = "CHANGE_MP"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeHPCommandBody struct {
	ChannelId byte  `json:"channelId"`
	Amount    int16 `json:"amount"`
}

type changeMPCommandBody struct {
	ChannelId byte  `json:"channelId"`
	Amount    int16 `json:"amount"`
}
