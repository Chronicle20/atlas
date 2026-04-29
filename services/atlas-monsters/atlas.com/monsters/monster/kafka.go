package monster

import (
	"encoding/json"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventMonsterStatusCreated          = "CREATED"
	EventMonsterStatusDestroyed        = "DESTROYED"
	EventMonsterStatusStartControl     = "START_CONTROL"
	EventMonsterStatusStopControl      = "STOP_CONTROL"
	EventMonsterStatusDamaged          = "DAMAGED"
	EventMonsterStatusKilled           = "KILLED"
	EventMonsterStatusEffectApplied    = "STATUS_APPLIED"
	EventMonsterStatusEffectExpired    = "STATUS_EXPIRED"
	EventMonsterStatusEffectCancelled  = "STATUS_CANCELLED"
	EventMonsterStatusDamageReflected  = "DAMAGE_REFLECTED"
	EventMonsterStatusFriendlyDrop     = "FRIENDLY_DROP"
	EventMonsterStatusAggroChanged     = "AGGRO_CHANGED"
	EventMonsterStatusNextSkillDecided = "NEXT_SKILL_DECIDED"

	DamageSourceCharacterAttack = "CHARACTER_ATTACK"
	DamageSourceMonsterAttack   = "MONSTER_ATTACK"
	DamageSourceDamageOverTime  = "DAMAGE_OVER_TIME"
	DamageSourceHeal            = "HEAL"
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
	ActorId            uint32 `json:"actorId"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Stance             byte   `json:"stance"`
	FH                 int16  `json:"fh"`
	Team               int8   `json:"team"`
	ControllerHasAggro bool   `json:"controllerHasAggro"`
}

type statusEventAggroChangedBody struct {
	ControllerCharacterId uint32 `json:"controllerCharacterId"`
	ControllerHasAggro    bool   `json:"controllerHasAggro"`
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
	DamageSource  string        `json:"damageSource"`
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
	ReflectKind       string           `json:"reflectKind"`
	ReflectPercent    int32            `json:"reflectPercent"`
	ReflectLtX        int16            `json:"reflectLtX"`
	ReflectLtY        int16            `json:"reflectLtY"`
	ReflectRbX        int16            `json:"reflectRbX"`
	ReflectRbY        int16            `json:"reflectRbY"`
	ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
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

type statusEventFriendlyDropBody struct {
	ItemCount uint32 `json:"itemCount"`
}

type statusEventNextSkillDecidedBody struct {
	SkillId                byte  `json:"skillId"`
	SkillLevel             byte  `json:"skillLevel"`
	DecidedAtMs            int64 `json:"decidedAtMs"`
	NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs"`
}

// MarshalJSON ensures DamageEntries marshals as `[]` rather than `null` when nil.
// See PRD FR-4.10 (cjson empty-array safety).
func (b statusEventDamagedBody) MarshalJSON() ([]byte, error) {
	type alias statusEventDamagedBody
	if b.DamageEntries == nil {
		b.DamageEntries = []damageEntry{}
	}
	return json.Marshal(alias(b))
}

// MarshalJSON ensures DamageEntries marshals as `[]` rather than `null` when nil.
func (b statusEventKilledBody) MarshalJSON() ([]byte, error) {
	type alias statusEventKilledBody
	if b.DamageEntries == nil {
		b.DamageEntries = []damageEntry{}
	}
	return json.Marshal(alias(b))
}

// MarshalJSON ensures Statuses marshals as `{}` rather than `null` when nil.
func (b statusEffectAppliedBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectAppliedBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}

// MarshalJSON ensures Statuses marshals as `{}` rather than `null` when nil.
func (b statusEffectExpiredBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectExpiredBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}

// MarshalJSON ensures Statuses marshals as `{}` rather than `null` when nil.
func (b statusEffectCancelledBody) MarshalJSON() ([]byte, error) {
	type alias statusEffectCancelledBody
	if b.Statuses == nil {
		b.Statuses = map[string]int32{}
	}
	return json.Marshal(alias(b))
}
