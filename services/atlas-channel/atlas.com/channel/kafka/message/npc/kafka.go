package npc

const (
	EnvCommandTopic                 = "COMMAND_TOPIC_NPC"
	CommandTypeStartConversation    = "START_CONVERSATION"
	CommandTypeContinueConversation = "CONTINUE_CONVERSATION"
	CommandTypeEndConversation      = "END_CONVERSATION"
)

type Command[E any] struct {
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StartConversationCommandBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type ContinueConversationCommandBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type EndConversationCommandBody struct {
}
