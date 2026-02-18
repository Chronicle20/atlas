package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventMonsterStatusCreated         = "CREATED"
	EventMonsterStatusDestroyed       = "DESTROYED"
	EventMonsterStatusStartControl    = "START_CONTROL"
	EventMonsterStatusStopControl     = "STOP_CONTROL"
	EventMonsterStatusDamaged         = "DAMAGED"
	EventMonsterStatusKilled          = "KILLED"
	EventMonsterStatusEffectApplied   = "STATUS_APPLIED"
	EventMonsterStatusEffectExpired   = "STATUS_EXPIRED"
	EventMonsterStatusEffectCancelled = "STATUS_CANCELLED"
	EventMonsterStatusDamageReflected = "DAMAGE_REFLECTED"
)

type statusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

func statusEventFromField[E any](f field.Model, uniqueId uint32, monsterId uint32, theType string, body E) statusEvent[E] {
	return statusEvent[E]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		UniqueId:  uniqueId,
		MonsterId: monsterId,
		Type:      theType,
		Body:      body,
	}
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
	ObserverId    uint32        `json:"observerId"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type statusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type damageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

type statusEffectAppliedBody struct {
	EffectId          string           `json:"effectId"`
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
}

type statusEffectExpiredBody struct {
	EffectId string           `json:"effectId"`
	Statuses map[string]int32 `json:"statuses"`
}

type statusEffectCancelledBody struct {
	EffectId string           `json:"effectId"`
	Statuses map[string]int32 `json:"statuses"`
}

type statusEventDamageReflectedBody struct {
	CharacterId   uint32 `json:"characterId"`
	ReflectDamage uint32 `json:"reflectDamage"`
	ReflectType   string `json:"reflectType"`
}
