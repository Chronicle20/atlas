package expression

const (
	EnvExpressionEvent = "EVENT_TOPIC_EXPRESSION"
)

type expressionEvent struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	Expression  uint32 `json:"expression"`
}
