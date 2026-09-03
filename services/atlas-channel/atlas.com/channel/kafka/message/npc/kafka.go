package npc

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_NPC"
)

const (
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
	Instance  uuid.UUID  `json:"instance"`
	AccountId uint32     `json:"accountId"`
}

type ContinueConversationCommandBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type EndConversationCommandBody struct{}

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_NPC_STATUS"
)

const (
	EventStatusTypeCreated = "CREATED"
)

// StatusEvent is the envelope for scripted-NPC lifecycle events consumed
// from EnvEventTopicStatus, mirroring atlas-maps'
// kafka/message/npc/kafka.go -- the producer side of this contract
// (task-290 task-BC) -- and this package's sibling monster StatusEvent
// shape (kafka/message/monster/kafka.go).
type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// CreatedStatusEventBody carries the placement of a newly spawned scripted
// NPC -- the fields atlas-maps' map/npc.Model holds (NpcId, X, Y, Fh).
type CreatedStatusEventBody struct {
	NpcId uint32 `json:"npcId"`
	X     int16  `json:"x"`
	Y     int16  `json:"y"`
	Fh    int16  `json:"fh"`
}
