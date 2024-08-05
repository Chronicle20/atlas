package monster

import "atlas-monsters/tenant"

const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"
	EnvCommandTopicDamage      = "COMMAND_TOPIC_MONSTER_DAMAGE"
	EnvCommandTopicMovement    = "COMMAND_TOPIC_MONSTER_MOVEMENT"

	consumerNameDamage   = "monster_damage_event"
	consumerNameMovement = "monster_movement_event"

	EventMonsterStatusCreated      = "CREATED"
	EventMonsterStatusDestroyed    = "DESTROYED"
	EventMonsterStatusStartControl = "START_CONTROL"
	EventMonsterStatusStopControl  = "STOP_CONTROL"
	EventMonsterStatusKilled       = "KILLED"
)

type statusEvent[E any] struct {
	Tenant    tenant.Model `json:"tenant"`
	WorldId   byte         `json:"worldId"`
	ChannelId byte         `json:"channelId"`
	MapId     uint32       `json:"mapId"`
	UniqueId  uint32       `json:"uniqueId"`
	MonsterId uint32       `json:"monsterId"`
	Type      string       `json:"type"`
	Body      E            `json:"body"`
}

type statusEventCreatedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventDestroyedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventStartControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventStopControlBody struct {
	ActorId uint32 `json:"actorId"`
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

type damageCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	WorldId     byte         `json:"worldId"`
	ChannelId   byte         `json:"channelId"`
	MapId       uint32       `json:"mapId"`
	UniqueId    uint32       `json:"uniqueId"`
	CharacterId uint32       `json:"characterId"`
	Damage      int64        `json:"damage"`
}

type movementCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	UniqueId      uint32       `json:"uniqueId"`
	ObserverId    int          `json:"observerId"`
	SkillPossible bool         `json:"skillPossible"`
	Skill         int          `json:"skill"`
	SkillId       int          `json:"skillId"`
	SkillLevel    int          `json:"skillLevel"`
	Option        int          `json:"option"`
	StartX        int          `json:"startX"`
	StartY        int          `json:"startY"`
	EndX          int16        `json:"endX"`
	EndY          int16        `json:"endY"`
	Stance        byte         `json:"stance"`
	RawMovement   []int        `json:"rawMovement"`
}
