package fame

const (
	EnvCommandTopic          = "COMMAND_TOPIC_FAME"
	CommandTypeRequestChange = "REQUEST_CHANGE"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestChangeCommandBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	TargetId  uint32 `json:"targetId"`
	Amount    int8   `json:"amount"`
}
