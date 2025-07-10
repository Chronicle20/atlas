package npc

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

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
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
}

type ContinueConversationCommandBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type EndConversationCommandBody struct {
}
