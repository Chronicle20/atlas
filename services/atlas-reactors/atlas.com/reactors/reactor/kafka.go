package reactor

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_REACTOR"
	CommandTypeCreate = "CREATE"
	CommandTypeHit    = "HIT"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreateCommandBody struct {
	Classification uint32 `json:"classification"`
	Name           string `json:"name"`
	State          int8   `json:"state"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Delay          uint32 `json:"delay"`
	Direction      byte   `json:"direction"`
}

type HitCommandBody struct {
	ReactorId   uint32 `json:"reactorId"`
	CharacterId uint32 `json:"characterId"`
	Stance      uint16 `json:"stance"`
	SkillId     uint32 `json:"skillId"`
}

// Reactor Actions topic and commands
const (
	EnvCommandReactorActionsTopic = "COMMAND_TOPIC_REACTOR_ACTIONS"
	CommandTypeActionsHit         = "HIT"
	CommandTypeActionsTrigger     = "TRIGGER"
)

// reactorActionsCommand represents a command sent to atlas-reactor-actions
type reactorActionsCommand[E any] struct {
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	Instance       uuid.UUID  `json:"instance"`
	ReactorId      uint32     `json:"reactorId"`
	Classification string     `json:"classification"`
	ReactorName    string     `json:"reactorName"`
	ReactorState   int8       `json:"reactorState"`
	X              int16      `json:"x"`
	Y              int16      `json:"y"`
	Type           string     `json:"type"`
	Body           E          `json:"body"`
}

// hitActionsBody represents the body of a HIT command to atlas-reactor-actions
type hitActionsBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	IsSkill     bool   `json:"isSkill"`
}

// triggerActionsBody represents the body of a TRIGGER command to atlas-reactor-actions
type triggerActionsBody struct {
	CharacterId uint32 `json:"characterId"`
}

const (
	EnvEventStatusTopic      = "EVENT_TOPIC_REACTOR_STATUS"
	EventStatusTypeCreated   = "CREATED"
	EventStatusTypeDestroyed = "DESTROYED"
	EventStatusTypeHit       = "HIT"
)

type statusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	ReactorId uint32     `json:"reactorId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

func statusEventFromField[E any](f field.Model, reactorId uint32, theType string, body E) statusEvent[E] {
	return statusEvent[E]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		ReactorId: reactorId,
		Type:      theType,
		Body:      body,
	}
}

type createdStatusEventBody struct {
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

type destroyedStatusEventBody struct {
	State int8  `json:"state"`
	X     int16 `json:"x"`
	Y     int16 `json:"y"`
}

type hitStatusEventBody struct {
	Classification uint32 `json:"classification"`
	State          int8   `json:"state"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Direction      byte   `json:"direction"`
	Destroyed      bool   `json:"destroyed"`
}
