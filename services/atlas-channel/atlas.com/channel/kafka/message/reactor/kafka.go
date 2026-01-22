package reactor

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_REACTOR"
	CommandTypeHit  = "HIT"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type HitCommandBody struct {
	ReactorId uint32 `json:"reactorId"`
	Stance    uint16 `json:"stance"`
	SkillId   uint32 `json:"skillId"`
}

const (
	EnvEventStatusTopic      = "EVENT_TOPIC_REACTOR_STATUS"
	EventStatusTypeCreated   = "CREATED"
	EventStatusTypeDestroyed = "DESTROYED"
	EventStatusTypeHit       = "HIT"
)

type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	ReactorId uint32     `json:"reactorId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreatedStatusEventBody struct {
	Classification uint32    `json:"classification"`
	Name           string    `json:"name"`
	State          int8      `json:"state"`
	EventState     byte      `json:"eventState"`
	Delay          uint32    `json:"delay"`
	Direction      byte      `json:"direction"`
	X              int16     `json:"x"`
	Y              int16     `json:"y"`
	UpdateTime     time.Time `json:"updateTime"`
}

type DestroyedStatusEventBody struct {
	State int8  `json:"state"`
	X     int16 `json:"x"`
	Y     int16 `json:"y"`
}

type HitStatusEventBody struct {
	Classification uint32 `json:"classification"`
	State          int8   `json:"state"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Direction      byte   `json:"direction"`
	Destroyed      bool   `json:"destroyed"`
}
