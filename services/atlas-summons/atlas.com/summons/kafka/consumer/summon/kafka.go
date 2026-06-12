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
