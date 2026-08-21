package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic           = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage         = "DAMAGE"
	CommandTypeDamageFriendly = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus    = "APPLY_STATUS"
	CommandTypeCancelStatus   = "CANCEL_STATUS"
	CommandTypeUseSkill       = "USE_SKILL"
	CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"
	CommandTypeDrainMp        = "DRAIN_MP"
	CommandTypeKill           = "KILL"
	CommandTypeClearAggro     = "CLEAR_AGGRO"
	CommandTypeForceControl   = "FORCE_CONTROL"
	CommandTypeBanish         = "BANISH"
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
	CharacterId uint32   `json:"characterId"`
	Damages     []uint32 `json:"damages"`
	AttackType  byte     `json:"attackType"`
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
	StatusTypes       []string `json:"statusTypes"`
	SourceCharacterId uint32   `json:"sourceCharacterId"`
	SourceSkillId     uint32   `json:"sourceSkillId"`
	// SourceSkillClass classifies the originating player skill as
	// "PHYSICAL" or "MAGICAL" (matching atlas-monsters'
	// monster.ReflectKind* constants). Empty when the cancel does not
	// originate from a player skill. atlas-monsters consults this to
	// refuse same-kind dispels of an active reflect (FR-4.9.1.2).
	SourceSkillClass string `json:"sourceSkillClass"`
}

type UseSkillCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     byte   `json:"skillId"`
	SkillLevel  byte   `json:"skillLevel"`
}

type UseBasicAttackCommandBody struct {
	AttackPos uint8 `json:"attackPos"`
}

// DrainMpCommandBody asks atlas-monsters to deduct MP from a monster
// because of a player passive. atlas-monsters re-checks Boss / MaxMp /
// current Mp guards and clamps the deduction at zero. On a non-zero
// drain it emits a MP_CHANGED status event with Reason set so the
// channel can refund the caster's MP and play the visual.
type DrainMpCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Amount      uint32 `json:"amount"`
}

// KillCommandBody asks atlas-monsters to kill a monster outright as the
// result of a player passive (Mortal Blow). The channel owns the threshold
// (hp ≤ maxHp·x/100) and kill-chance (roll ≤ y) decisions; atlas-monsters
// re-checks alive + boss (fail-closed) and delivers the kill through the
// standard damage path so EXP and drops credit the attacker like a normal
// kill.
//
// No skillId is carried: atlas-monsters never resolves it, the channel
// already logs it locally at proc time, and the trace id correlates the
// two services. Critically, the monster command topic fans every message
// to every registered handler, each json-unmarshalling the body into its
// own type; a large job-skill id (e.g. 3110001) would overflow the
// byte-typed skillId in sibling bodies (useSkillCommandBody) and log a
// spurious unmarshal error per proc. Keep this body minimal.
type KillCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// ClearAggroCommandBody asks atlas-monsters to fully wipe a monster's
// accumulated damage-aggro table — every character's entry, not a decay toward
// the aggro floor. Deliberately EMPTY (FR-4.3): the command is orthogonal and
// carries nothing magnet-specific, and an empty body cannot collide with a
// sibling body's field types on this shared, fan-to-every-handler topic.
type ClearAggroCommandBody struct{}

// ForceControlCommandBody asks atlas-monsters to hand a monster's controller to
// a named character, bypassing the normal picker election, and to set the
// controller-has-aggro flag so the resulting START_CONTROL drives
// writer.StartControlMonsterBody(m, true).
//
// characterId is the only field, and `characterId uint32` already appears with
// that exact name and type in DamageCommandBody, KillCommandBody and the
// catch body, so it introduces no unmarshal collision on the shared topic.
type ForceControlCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// BanishCommandBody asks atlas-monsters to banish a character out of a field on
// the strength of a client MOB_BANISH_PLAYER request. MonsterTemplateId is
// client-supplied and untrusted; atlas-monsters revalidates it against live
// field state before acting. Both fields are uint32: characterId already
// appears at that name and type in DamageCommandBody / KillCommandBody /
// ForceControlCommandBody, and monsterTemplateId appears in no sibling body, so
// neither can collide on the shared, fan-to-every-handler command topic. The
// envelope's monsterId is deliberately left 0 — it means *unique* id everywhere
// else on this topic. Mirrors atlas-monsters' banishCommandBody — edit both
// together.
type BanishCommandBody struct {
	CharacterId       uint32 `json:"characterId"`
	MonsterTemplateId uint32 `json:"monsterTemplateId"`
}

const (
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventStatusCreated          = "CREATED"
	EventStatusDestroyed        = "DESTROYED"
	EventStatusStartControl     = "START_CONTROL"
	EventStatusStopControl      = "STOP_CONTROL"
	EventStatusDamaged          = "DAMAGED"
	EventStatusKilled           = "KILLED"
	EventStatusEffectApplied    = "STATUS_APPLIED"
	EventStatusEffectExpired    = "STATUS_EXPIRED"
	EventStatusEffectCancelled  = "STATUS_CANCELLED"
	EventStatusDamageReflected  = "DAMAGE_REFLECTED"
	EventStatusAggroChanged     = "AGGRO_CHANGED"
	EventStatusNextSkillDecided = "NEXT_SKILL_DECIDED"
	EventStatusMpChanged        = "MP_CHANGED"
	EventStatusCaught           = "CAUGHT"
	EventStatusCatchFailed      = "CATCH_FAILED"

	// CatchCauseSpeciesMismatch / CatchCauseHpTooHigh / CatchCauseRollFailed /
	// CatchCauseUnresolved are the internal failure causes atlas-monsters emits
	// on CATCH_FAILED. The mapping onto the client's wire reason byte is owned
	// here in atlas-channel (DOM-25) -- see bridleFailReason in
	// kafka/consumer/monster/consumer.go.
	CatchCauseSpeciesMismatch = "SPECIES_MISMATCH"
	CatchCauseHpTooHigh       = "HP_TOO_HIGH"
	CatchCauseRollFailed      = "ROLL_FAILED"
	CatchCauseUnresolved      = "UNRESOLVED"

	DamageSourceCharacterAttack = "CHARACTER_ATTACK"
	DamageSourceMonsterAttack   = "MONSTER_ATTACK"
	DamageSourceDamageOverTime  = "DAMAGE_OVER_TIME"
	DamageSourceHeal            = "HEAL"

	MpChangeReasonMpEater     = "MP_EATER"
	MpChangeReasonSkillCast   = "SKILL_CAST"
	MpChangeReasonBasicAttack = "BASIC_ATTACK"
	MpChangeReasonRecovery    = "RECOVERY"
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
	ActorId            uint32 `json:"actorId"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Stance             byte   `json:"stance"`
	FH                 int16  `json:"fh"`
	Team               int8   `json:"team"`
	ControllerHasAggro bool   `json:"controllerHasAggro"`
}

type StatusEventStopControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type StatusEventDamagedBody struct {
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	ObserverId uint32 `json:"observerId"`
	ActorId    uint32 `json:"actorId"`
	Boss       bool   `json:"boss"`
	// Damage is the amount THIS event applied. DamageEntries is the monster's
	// running per-character total (kill credit / drop ownership) -- reading its
	// last element as "the damage" reports a cumulative figure, which is what
	// this field exists to prevent.
	Damage        uint32        `json:"damage"`
	DamageSource  string        `json:"damageSource"`
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
	ReflectKind       string           `json:"reflectKind"`
	ReflectPercent    int32            `json:"reflectPercent"`
	ReflectLtX        int16            `json:"reflectLtX"`
	ReflectLtY        int16            `json:"reflectLtY"`
	ReflectRbX        int16            `json:"reflectRbX"`
	ReflectRbY        int16            `json:"reflectRbY"`
	ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
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

type StatusEventAggroChangedBody struct {
	ControllerCharacterId uint32 `json:"controllerCharacterId"`
	ControllerHasAggro    bool   `json:"controllerHasAggro"`
}

type StatusEventNextSkillDecidedBody struct {
	SkillId                byte  `json:"skillId"`
	SkillLevel             byte  `json:"skillLevel"`
	DecidedAtMs            int64 `json:"decidedAtMs"`
	NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs"`
}

// StatusEventMpChangedBody is the return event for any monster MP
// mutation whose Reason atlas-channel needs to react to. v1 only emits
// Reason = MpChangeReasonMpEater; future passives (e.g., Magic Guard
// refund, Drain MP) will share the channel by setting a new Reason.
type StatusEventMpChangedBody struct {
	CharacterId    uint32 `json:"characterId"`
	SkillId        uint32 `json:"skillId"`
	Reason         string `json:"reason"`
	Amount         uint32 `json:"amount"`
	MonsterMpAfter uint32 `json:"monsterMpAfter"`
}

// StatusEventCaughtBody carries the successful outcome of a bridle
// (taming-item) capture attempt.
type StatusEventCaughtBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

// StatusEventCatchFailedBody carries a failed bridle capture attempt. Cause
// is one of the Catch cause constants above; the wire-reason mapping is
// resolved in atlas-channel, never emitted by atlas-monsters (DOM-25).
type StatusEventCatchFailedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Cause       string `json:"cause"`
}
