package npc

const (
	EnvCommandTopic              = "COMMAND_TOPIC_NPC"
	CommandTypeStartConversation = "START_CONVERSATION"
)

type command[E any] struct {
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type startConversationCommandBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type continueConversationCommandBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type endConversationCommandBody struct {
}
