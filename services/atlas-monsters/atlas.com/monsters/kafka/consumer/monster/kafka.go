package monster

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic              = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage            = "DAMAGE"
	CommandTypeDamageFriendly    = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus       = "APPLY_STATUS"
	CommandTypeCancelStatus      = "CANCEL_STATUS"
	CommandTypeUseSkill          = "USE_SKILL"
	CommandTypeApplyStatusField  = "APPLY_STATUS_FIELD"
	CommandTypeCancelStatusField = "CANCEL_STATUS_FIELD"
	CommandTypeUseSkillField     = "USE_SKILL_FIELD"
	CommandTypeDestroyField      = "DESTROY_FIELD"

	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)

type command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
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

type destroyFieldCommandBody struct{}

type movementCommand struct {
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      _map.Id    `json:"mapId"`
	Instance   uuid.UUID  `json:"instance"`
	ObjectId   uint64     `json:"objectId"`
	ObserverId uint32     `json:"observerId"`
	X          int16      `json:"x"`
	Y          int16      `json:"y"`
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
