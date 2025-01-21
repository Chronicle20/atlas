package npc

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"

	EnvCommandTopic   = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple = "SIMPLE"
	CommandTypeText   = "TEXT"
	CommandTypeStyle  = "STYLE"
	CommandTypeNumber = "NUMBER"
)

type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

// TODO this should transmit stats
type statusEventStatChangedBody struct {
	ChannelId       byte `json:"channelId"`
	ExclRequestSent bool `json:"exclRequestSent"`
}

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	NpcId       uint32 `json:"npcId"`
	Speaker     string `json:"speaker"`
	Message     string `json:"message"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type commandSimpleBody struct {
	Type string `json:"type"`
}
