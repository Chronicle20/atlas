package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage = "DAMAGE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type DamageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

const (
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventStatusCreated      = "CREATED"
	EventStatusDestroyed    = "DESTROYED"
	EventStatusStartControl = "START_CONTROL"
	EventStatusStopControl  = "STOP_CONTROL"
	EventStatusDamaged      = "DAMAGED"
	EventStatusKilled       = "KILLED"
)

type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type StatusEventCreatedBody struct {
	ActorId uint32 `json:"actorId"`
}

type StatusEventDestroyedBody struct {
	ActorId uint32 `json:"actorId"`
}

type StatusEventStartControlBody struct {
	ActorId uint32 `json:"actorId"`
	X       int16  `json:"x"`
	Y       int16  `json:"y"`
	Stance  byte   `json:"stance"`
	FH      int16  `json:"fh"`
	Team    int8   `json:"team"`
}

type StatusEventStopControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type StatusEventDamagedBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type StatusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      int64  `json:"damage"`
}
