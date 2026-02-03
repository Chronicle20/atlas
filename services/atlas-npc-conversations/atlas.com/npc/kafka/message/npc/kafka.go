package npc

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

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
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	AccountId uint32     `json:"accountId"`
}

type CommandConversationContinueBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type CommandConversationEndBody struct {
}

type ConversationCommand[E any] struct {
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	Instance       uuid.UUID  `json:"instance"`
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

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

// TODO this should transmit stats
type StatusEventStatChangedBody struct {
	ChannelId       channel.Id `json:"channelId"`
	ExclRequestSent bool       `json:"exclRequestSent"`
}
