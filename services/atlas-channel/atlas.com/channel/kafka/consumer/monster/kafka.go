package monster

import "atlas-channel/movement"

const (
	EnvEventTopicStatus   = "EVENT_TOPIC_MONSTER_STATUS"
	EnvEventTopicMovement = "EVENT_TOPIC_MONSTER_MOVEMENT"

	EventStatusCreated      = "CREATED"
	EventStatusDestroyed    = "DESTROYED"
	EventStatusStartControl = "START_CONTROL"
	EventStatusStopControl  = "STOP_CONTROL"
	EventStatusDamaged      = "DAMAGED"
	EventStatusKilled       = "KILLED"
)

type statusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	UniqueId  uint32 `json:"uniqueId"`
	MonsterId uint32 `json:"monsterId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type statusEventCreatedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventDestroyedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventStartControlBody struct {
	ActorId uint32 `json:"actorId"`
	X       int16  `json:"x"`
	Y       int16  `json:"y"`
	Stance  byte   `json:"stance"`
	FH      int16  `json:"fh"`
	Team    int8   `json:"team"`
}

type statusEventStopControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventDamagedBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type statusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type damageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      int64  `json:"damage"`
}

type movementEvent struct {
	WorldId       byte              `json:"worldId"`
	ChannelId     byte              `json:"channelId"`
	MapId         uint32            `json:"mapId"`
	UniqueId      uint32            `json:"uniqueId"`
	ObserverId    uint32            `json:"observerId"`
	SkillPossible bool              `json:"skillPossible"`
	Skill         int8              `json:"skill"`
	SkillId       int16             `json:"skillId"`
	SkillLevel    int16             `json:"skillLevel"`
	MultiTarget   []position        `json:"multiTarget"`
	RandomTimes   []int32           `json:"randomTimes"`
	Movement      movement.Movement `json:"movement"`
}

type position struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}
