package chalkboard

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic        = "COMMAND_TOPIC_CHALKBOARD"
	CommandChalkboardSet   = "SET"
	CommandChalkboardClear = "CLEAR"
)

type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type SetCommandBody struct {
	Message string `json:"message"`
}

type ClearCommandBody struct {
}

const (
	EnvEventTopicStatus       = "EVENT_TOPIC_CHALKBOARD_STATUS"
	EventTopicStatusTypeSet   = "SET"
	EventTopicStatusTypeClear = "CLEAR"
)

type StatusEvent[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type SetStatusEventBody struct {
	Message string `json:"message"`
}

type ClearStatusEventBody struct {
}
