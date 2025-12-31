package conversation

const (
	EnvCommandTopic   = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple = "SIMPLE"
	CommandTypeText   = "TEXT"
	CommandTypeStyle  = "STYLE"
	CommandTypeNumber = "NUMBER"
)

type CommandEvent[E any] struct {
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

type CommandNumberBody struct {
	DefaultValue uint32 `json:"defaultValue"`
	MinValue     uint32 `json:"minValue"`
	MaxValue     uint32 `json:"maxValue"`
}
