package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic           = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage         = "DAMAGE"
	CommandTypeDamageFriendly = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus    = "APPLY_STATUS"
	CommandTypeCancelStatus   = "CANCEL_STATUS"
	CommandTypeUseSkill       = "USE_SKILL"
)

type DamageFriendlyCommandBody struct {
	AttackerUniqueId uint32 `json:"attackerUniqueId"`
	ObserverUniqueId uint32 `json:"observerUniqueId"`
}

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
	AttackType  byte   `json:"attackType"`
}

type ApplyStatusCommandBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

type CancelStatusCommandBody struct {
	StatusTypes []string `json:"statusTypes"`
}

type UseSkillCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint16 `json:"skillId"`
	SkillLevel  uint16 `json:"skillLevel"`
}

const (
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventStatusCreated         = "CREATED"
	EventStatusDestroyed       = "DESTROYED"
	EventStatusStartControl    = "START_CONTROL"
	EventStatusStopControl     = "STOP_CONTROL"
	EventStatusDamaged         = "DAMAGED"
	EventStatusKilled          = "KILLED"
	EventStatusEffectApplied   = "STATUS_APPLIED"
	EventStatusEffectExpired   = "STATUS_EXPIRED"
	EventStatusEffectCancelled = "STATUS_CANCELLED"
	EventStatusDamageReflected = "DAMAGE_REFLECTED"
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
	ObserverId    uint32        `json:"observerId"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type StatusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []DamageEntry `json:"damageEntries"`
}

type DamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      int64  `json:"damage"`
}

type StatusEffectAppliedBody struct {
	EffectId          string           `json:"effectId"`
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
}

type StatusEffectExpiredBody struct {
	EffectId string           `json:"effectId"`
	Statuses map[string]int32 `json:"statuses"`
}

type StatusEffectCancelledBody struct {
	EffectId string           `json:"effectId"`
	Statuses map[string]int32 `json:"statuses"`
}

type StatusEventDamageReflectedBody struct {
	CharacterId   uint32 `json:"characterId"`
	ReflectDamage uint32 `json:"reflectDamage"`
	ReflectType   string `json:"reflectType"`
}
