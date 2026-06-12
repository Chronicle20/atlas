package summon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvCommandTopic = "COMMAND_TOPIC_SUMMON"

const (
	CommandTypeSpawn  = "SPAWN"
	CommandTypeMove   = "MOVE"
	CommandTypeAttack = "ATTACK"
	CommandTypeDamage = "DAMAGE"
)

// Command is the COMMAND_TOPIC_SUMMON envelope (channel -> summons).
type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	SummonId  uint32     `json:"summonId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type SpawnCommandBody struct {
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	SkillId          uint32 `json:"skillId"`
	SkillLevel       byte   `json:"skillLevel"`
	X                int16  `json:"x"`
	Y                int16  `json:"y"`
	// AuraLevel/HexLevel carry the caster's trained AURA_OF_THE_BEHOLDER (1320008)
	// and HEX_OF_THE_BEHOLDER (1320009) levels for a Beholder summon; 0 for all
	// other summons. The channel resolves them from the caster's skill book.
	AuraLevel byte `json:"auraLevel"`
	HexLevel  byte `json:"hexLevel"`
}

type MoveCommandBody struct {
	SummonId          uint32 `json:"summonId"`
	SenderCharacterId uint32 `json:"senderCharacterId"`
	X                 int16  `json:"x"`
	Y                 int16  `json:"y"`
	Stance            byte   `json:"stance"`
	RawMovement       []byte `json:"rawMovement"`
}

// AttackTargetEntry is one {monster, reported damage} pair carried by an ATTACK
// command. The damage is the raw client-reported value; atlas-summons clamps it.
type AttackTargetEntry struct {
	MonsterId uint32 `json:"monsterId"`
	Damage    uint32 `json:"damage"`
}

type AttackCommandBody struct {
	SummonId          uint32              `json:"summonId"`
	SenderCharacterId uint32              `json:"senderCharacterId"`
	Direction         byte                `json:"direction"`
	Targets           []AttackTargetEntry `json:"targets"`
}

// DamageCommandBody carries a monster-dealt damage report against a puppet
// summon. atlas-summons verifies the summon exists, decrements its HP, and
// emits a DAMAGED event (destroying the summon at zero HP).
type DamageCommandBody struct {
	SummonId          uint32 `json:"summonId"`
	SenderCharacterId uint32 `json:"senderCharacterId"`
	Damage            int32  `json:"damage"`
	MonsterIdFrom     uint32 `json:"monsterIdFrom"`
}
