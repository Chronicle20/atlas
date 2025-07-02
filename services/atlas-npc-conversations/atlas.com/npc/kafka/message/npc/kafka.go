package npc

const (
	EnvCommandTopic                 = "COMMAND_TOPIC_NPC"
	CommandTypeStartConversation    = "START_CONVERSATION"
	CommandTypeContinueConversation = "CONTINUE_CONVERSATION"
	CommandTypeEndConversation      = "END_CONVERSATION"

	EnvConversationCommandTopic = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple           = "SIMPLE"
	CommandTypeText             = "TEXT"
	CommandTypeStyle            = "STYLE"
	CommandTypeNumber           = "NUMBER"
)

type Command[E any] struct {
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CommandConversationStartBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type CommandConversationContinueBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type CommandConversationEndBody struct {
}

type ConversationCommand[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	NpcId       uint32 `json:"npcId"`
	Speaker     string `json:"speaker"`
	Message     string `json:"message"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CommandSimpleBody struct {
	Type string `json:"type"`
}

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

// TODO this should transmit stats
type StatusEventStatChangedBody struct {
	ChannelId       byte `json:"channelId"`
	ExclRequestSent bool `json:"exclRequestSent"`
}
