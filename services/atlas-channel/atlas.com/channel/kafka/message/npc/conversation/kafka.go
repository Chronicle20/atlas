package conversation

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple = "SIMPLE"
	CommandTypeText   = "TEXT"
	CommandTypeStyle  = "STYLE"
	CommandTypeNumber = "NUMBER"
)

type CommandEvent[E any] struct {
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	CharacterId    uint32     `json:"characterId"`
	NpcId          uint32     `json:"npcId"`
	Speaker        string     `json:"speaker"`
	EndChat        bool       `json:"endChat"`
	SecondaryNpcId uint32     `json:"secondaryNpcId"`
	Message        string     `json:"message"`
	Type           string     `json:"type"`
	Body           E          `json:"body"`
}

type CommandSimpleBody struct {
	Type string `json:"type"`
}

type CommandNumberBody struct {
	DefaultValue uint32 `json:"defaultValue"`
	MinValue     uint32 `json:"minValue"`
	MaxValue     uint32 `json:"maxValue"`
}

type CommandStyleBody struct {
	Styles []uint32 `json:"styles"`
}
