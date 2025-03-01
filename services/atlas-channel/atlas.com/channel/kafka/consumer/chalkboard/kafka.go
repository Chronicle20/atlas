package chalkboard

const (
	EnvEventTopicStatus       = "EVENT_TOPIC_CHALKBOARD_STATUS"
	EventTopicStatusTypeSet   = "SET"
	EventTopicStatusTypeClear = "CLEAR"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type setStatusEventBody struct {
	Message string `json:"message"`
}

type clearStatusEventBody struct {
}
