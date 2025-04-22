package chalkboard

const (
	EnvCommandTopic        = "COMMAND_TOPIC_CHALKBOARD"
	CommandChalkboardSet   = "SET"
	CommandChalkboardClear = "CLEAR"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type SetCommandBody struct {
	Message string `json:"message"`
}

type ClearCommandBody struct {
}

const (
	EnvEventTopicStatus       = "EVENT_TOPIC_CHALKBOARD_STATUS"
	EventTopicStatusTypeSet   = "SET"
	EventTopicStatusTypeClear = "CLEAR"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type SetStatusEventBody struct {
	Message string `json:"message"`
}

type ClearStatusEventBody struct {
}
