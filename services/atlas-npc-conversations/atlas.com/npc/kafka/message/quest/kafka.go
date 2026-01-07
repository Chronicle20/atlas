package quest

const (
	EnvCommandTopic                   = "COMMAND_TOPIC_QUEST_CONVERSATION"
	CommandTypeStartQuestConversation = "START_QUEST_CONVERSATION"
)

type Command[E any] struct {
	QuestId     uint32 `json:"questId"`
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StartQuestConversationCommandBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}
