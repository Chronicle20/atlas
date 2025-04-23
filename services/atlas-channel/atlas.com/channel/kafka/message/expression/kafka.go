package expression

const (
	EnvExpressionCommand = "COMMAND_TOPIC_EXPRESSION"
)

type Command struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	Expression  uint32 `json:"expression"`
}

const (
	EnvExpressionEvent = "EVENT_TOPIC_EXPRESSION"
)

type Event struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	Expression  uint32 `json:"expression"`
}
