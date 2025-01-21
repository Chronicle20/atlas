package npc

const (
	EnvCommandTopic              = "COMMAND_TOPIC_NPC"
	CommandTypeStartConversation = "START_CONVERSATION"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type startConversationCommandBody struct {
}
