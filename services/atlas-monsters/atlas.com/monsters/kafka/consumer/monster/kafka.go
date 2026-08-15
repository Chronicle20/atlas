package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic              = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage            = "DAMAGE"
	CommandTypeDamageFriendly    = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus       = "APPLY_STATUS"
	CommandTypeCancelStatus      = "CANCEL_STATUS"
	CommandTypeUseSkill          = "USE_SKILL"
	CommandTypeUseBasicAttack    = "USE_BASIC_ATTACK"
	CommandTypeApplyStatusField  = "APPLY_STATUS_FIELD"
	CommandTypeCancelStatusField = "CANCEL_STATUS_FIELD"
	CommandTypeUseSkillField     = "USE_SKILL_FIELD"
	CommandTypeDestroyField      = "DESTROY_FIELD"
	CommandTypeSpawnField        = "SPAWN_FIELD"
	CommandTypeDrainMp           = "DRAIN_MP"
	CommandTypeAddPuppet         = "ADD_PUPPET"
	CommandTypeRemovePuppet      = "REMOVE_PUPPET"
	CommandTypeKill              = "KILL"
	CommandTypeCatch             = "CATCH"
	CommandTypeClearAggro        = "CLEAR_AGGRO"
	CommandTypeForceControl      = "FORCE_CONTROL"

	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)

type command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type damageFriendlyCommandBody struct {
	ObserverUniqueId uint32 `json:"observerUniqueId"`
	AttackerUniqueId uint32 `json:"attackerUniqueId"`
}

type damageCommandBody struct {
	CharacterId uint32   `json:"characterId"`
	Damages     []uint32 `json:"damages"`
	AttackType  byte     `json:"attackType"`
}

type applyStatusCommandBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

type cancelStatusCommandBody struct {
	StatusTypes       []string `json:"statusTypes"`
	SourceCharacterId uint32   `json:"sourceCharacterId"`
	SourceSkillId     uint32   `json:"sourceSkillId"`
	// SourceSkillClass classifies a player-originated cancel as "PHYSICAL"
	// or "MAGICAL" (matching monster.ReflectKind* constants), or is empty
	// when the cancel originates internally (expiry, mutual exclusion).
	// The processor consults it to gate dispels against same-kind reflects
	// (FR-4.9.1.2).
	SourceSkillClass string `json:"sourceSkillClass"`
}

type useSkillCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     byte   `json:"skillId"`
	SkillLevel  byte   `json:"skillLevel"`
}

type useSkillFieldCommandBody struct {
	SkillId    byte `json:"skillId"`
	SkillLevel byte `json:"skillLevel"`
}

type useBasicAttackCommandBody struct {
	AttackPos uint8 `json:"attackPos"`
}

type destroyFieldCommandBody struct{}

// spawnFieldCommandBody carries optional spawn provenance (FR-P1). Both field
// names are absent from every sibling body on this shared, fan-to-every-handler
// topic, so adding them cannot produce a spurious unmarshal error the way a
// type-mismatched name would (see killCommandBody's note). omitempty keeps an
// omitting producer's bytes identical to today (FR-P5).
type spawnFieldCommandBody struct {
	MonsterId       uint32 `json:"monsterId"`
	X               int16  `json:"x"`
	Y               int16  `json:"y"`
	Fh              int16  `json:"fh"`
	Team            int8   `json:"team"`
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
}

type drainMpCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Amount      uint32 `json:"amount"`
}

// killCommandBody asks the processor to kill a monster outright as the
// result of a player passive (Mortal Blow). The channel already rolled
// the threshold and kill chance; the processor re-checks alive + boss
// (fail-closed).
//
// No skillId field: it is never resolved here, and because every handler
// on this shared command topic json-unmarshals every message, a large
// job-skill id would overflow the byte-typed skillId in sibling bodies
// (useSkillCommandBody) and log a spurious unmarshal error per proc.
type killCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// catchCommandBody asks the processor to remove a monster as a bridle
// (catch-item) capture. Deliberately minimal: every handler on this shared
// command topic json-unmarshals every message, so a field name whose type
// disagrees with a sibling body produces one spurious unmarshal error per
// message. characterId and itemId are both uint32 and both already appear with
// that type in sibling bodies (damageCommandBody.CharacterId,
// drainMpCommandBody.SkillId).
type catchCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

// clearAggroCommandBody asks the processor to fully wipe a monster's
// damage-aggro table. Deliberately empty: the command carries nothing
// caller-specific, and an empty body cannot collide with a sibling body's field
// types on this shared, fan-to-every-handler topic. Mirrors
// atlas-channel's monster2.ClearAggroCommandBody — edit both together.
type clearAggroCommandBody struct{}

// forceControlCommandBody asks the processor to hand control of a monster to a
// named character with the aggro flag set, bypassing the picker. Mirrors
// atlas-channel's monster2.ForceControlCommandBody — edit both together.
type forceControlCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// addPuppetCommand registers a player's puppet in a field so the monster
// controller picker can bias toward the puppet's owner. Emitted by atlas-summons
// on puppet spawn. Type must equal CommandTypeAddPuppet.
type addPuppetCommand struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	Type             string     `json:"type"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int16      `json:"x"`
	Y                int16      `json:"y"`
}

// removePuppetCommand clears a previously registered puppet. Emitted by
// atlas-summons on puppet despawn. Type must equal CommandTypeRemovePuppet.
type removePuppetCommand struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	Type             string     `json:"type"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
}

type movementCommand struct {
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      _map.Id    `json:"mapId"`
	Instance   uuid.UUID  `json:"instance"`
	ObjectId   uint64     `json:"objectId"`
	ObserverId uint32     `json:"observerId"`
	X          int16      `json:"x"`
	Y          int16      `json:"y"`
	Fh         int16      `json:"fh"`
	Stance     byte       `json:"stance"`
}

type fieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}
