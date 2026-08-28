package monster

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicMonsterStatus topic.Token = "EVENT_TOPIC_MONSTER_STATUS"
	EnvEventTopicMonsterCatch  topic.Token = "EVENT_TOPIC_MONSTER_CATCH"

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
	EventMonsterStatusMpChanged        = "MP_CHANGED"
	EventMonsterStatusCaught           = "CAUGHT"
	EventMonsterStatusCatchFailed      = "CATCH_FAILED"

	EventMonsterCatchResolved = "CATCH_RESOLVED"

	DamageSourceCharacterAttack = "CHARACTER_ATTACK"
	DamageSourceMonsterAttack   = "MONSTER_ATTACK"
	DamageSourceDamageOverTime  = "DAMAGE_OVER_TIME"
	DamageSourceHeal            = "HEAL"

	MpChangeReasonMpEater     = "MP_EATER"
	MpChangeReasonSkillCast   = "SKILL_CAST"
	MpChangeReasonBasicAttack = "BASIC_ATTACK"
	MpChangeReasonRecovery    = "RECOVERY"

	// Catch failure causes. The wire collapses all of them to a single byte
	// (design §6.4), so these survive only in logs and in the channel's
	// cause -> wire-reason mapping. UNRESOLVED means the attempt lost a race
	// (monster gone, or another catcher claimed it) — the channel renders no
	// failure packet for it, only the unlock.
	CatchCauseSpeciesMismatch = "SPECIES_MISMATCH"
	CatchCauseHpTooHigh       = "HP_TOO_HIGH"
	CatchCauseRollFailed      = "ROLL_FAILED"
	CatchCauseUnresolved      = "UNRESOLVED"

	// DeathType* are the semantic keys atlas-channel resolves through the
	// tenant's `operations` writer-options table for the DestroyMonster writer
	// (DOM-25) -- the same pattern as CatchCause* above. They map 1:1 onto the
	// closed DestroyType enum in libs/atlas-packet monster/clientbound
	// (CMob::m_nDeadType). DeathTypeUnset is what an omitting producer sends
	// (rolling deploy); every consumer treats the empty string as fade-out, so
	// the wire stays byte-identical to pre-task-253 behaviour. DeathTypeUnset
	// is a documentation alias for "" — the JSON zero value for a string field
	// — not a distinct wire state.
	DeathTypeUnset          = ""
	DeathTypeDisappear      = "DISAPPEAR"
	DeathTypeFadeOut        = "FADE_OUT"
	DeathTypeBomb           = "BOMB"
	DeathTypeDestructByMiss = "DESTRUCT_BY_MISS"
	DeathTypeSwallow        = "SWALLOW"
	DeathTypeSelfDestruct   = "SELF_DESTRUCT"
)

type statusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	// SpawnSourceType / SpawnSourceId echo the monster's provenance on EVERY
	// status type (FR-P3 requires CREATED/KILLED/DESTROYED; the envelope gives
	// the superset for free and makes forgetting one impossible). omitempty so
	// a cyclic monster's events are byte-identical to today.
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
	Body            E      `json:"body"`
}

func statusEventFromField[E any](f field.Model, uniqueId uint32, monsterId uint32, theType string, body E, spawnSourceType string, spawnSourceId string) statusEvent[E] {
	return statusEvent[E]{
		WorldId:         f.WorldId(),
		ChannelId:       f.ChannelId(),
		MapId:           f.MapId(),
		Instance:        f.Instance(),
		UniqueId:        uniqueId,
		MonsterId:       monsterId,
		Type:            theType,
		SpawnSourceType: spawnSourceType,
		SpawnSourceId:   spawnSourceId,
		Body:            body,
	}
}

type statusEventCreatedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventDestroyedBody struct {
	ActorId   uint32 `json:"actorId"`
	DeathType string `json:"deathType"`
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
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	ObserverId uint32 `json:"observerId"`
	ActorId    uint32 `json:"actorId"`
	Boss       bool   `json:"boss"`
	// Damage is the amount THIS event applied. DamageEntries below is the
	// monster's running per-character total and answers a different question
	// (who gets the kill credit / drop ownership); a consumer that wants the
	// number to render must read Damage, never the last DamageEntries element.
	Damage        uint32        `json:"damage"`
	DamageSource  string        `json:"damageSource"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type statusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	Boss          bool          `json:"boss"`
	DamageEntries []damageEntry `json:"damageEntries"`
	DeathType     string        `json:"deathType"`
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

type statusEventMpChangedBody struct {
	CharacterId    uint32 `json:"characterId"`
	SkillId        uint32 `json:"skillId"`
	Reason         string `json:"reason"`
	Amount         uint32 `json:"amount"`
	MonsterMpAfter uint32 `json:"monsterMpAfter"`
}

// catchResolvedBody is the economic outcome, published on the dedicated
// low-volume EVENT_TOPIC_MONSTER_CATCH. atlas-consumables consumes it to commit
// or cancel the item reservation; it must NOT be published on the status topic,
// which carries a DAMAGED event per hit and whose every handler unmarshals every
// message (design §4.2).
type catchResolvedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Success     bool   `json:"success"`
	Cause       string `json:"cause"`
}

// statusEventCaughtBody is the presentation outcome, published on the status
// topic immediately BEFORE DESTROYED. The status topic is keyed by MapId, so
// that ordering is a partition guarantee — which matters because
// CMobPool::OnMobPacket resolves the mob via GetMob and silently drops the
// effect packet if the mob is already gone.
type statusEventCaughtBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

type statusEventCatchFailedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Cause       string `json:"cause"`
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
