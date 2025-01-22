package conversation

const (
	EnvCommandTopic   = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple = "SIMPLE"
	CommandTypeText   = "TEXT"
	CommandTypeStyle  = "STYLE"
	CommandTypeNumber = "NUMBER"
)

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
