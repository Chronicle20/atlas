package summon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// EnvCommandTopic is the COMMAND_TOPIC_SUMMON env var (channel -> summons).
// The envelope and body below are re-declared channel-side; their JSON tags
// MUST stay byte-for-byte identical to the atlas-summons consumer definition
// at services/atlas-summons/atlas.com/summons/kafka/consumer/summon/kafka.go.
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
// command. JSON tags MUST stay byte-for-byte identical to the atlas-summons
// consumer definition.
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

// EnvEventTopicSummonStatus is the EVENT_TOPIC_SUMMON_STATUS env var
// (summons -> channel). The envelope and bodies below are re-declared
// channel-side; their JSON tags MUST stay byte-for-byte identical to the
// atlas-summons producer definition at
// services/atlas-summons/atlas.com/summons/summon/kafka.go.
const EnvEventTopicSummonStatus = "EVENT_TOPIC_SUMMON_STATUS"

const (
	EventSummonStatusCreated   = "CREATED"
	EventSummonStatusMoved     = "MOVED"
	EventSummonStatusAttacked  = "ATTACKED"
	EventSummonStatusDamaged   = "DAMAGED"
	EventSummonStatusDestroyed = "DESTROYED"
)

// StatusEvent is the summon-status event envelope consumed from atlas-summons.
type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	SummonId         uint32     `json:"summonId"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	SkillId          uint32     `json:"skillId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type StatusEventCreatedBody struct {
	SkillLevel   byte  `json:"skillLevel"`
	MovementType byte  `json:"movementType"`
	X            int16 `json:"x"`
	Y            int16 `json:"y"`
	Stance       byte  `json:"stance"`
	Puppet       bool  `json:"puppet"`
	Animated     bool  `json:"animated"`
}

type StatusEventMovedBody struct {
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	Stance      byte   `json:"stance"`
	RawMovement []byte `json:"rawMovement"`
}

type StatusEventDestroyedBody struct {
	Animated bool `json:"animated"`
}

// StatusEventAttackedTarget is one {monster, clamped damage} pair carried by an
// ATTACKED event. JSON tags MUST stay byte-for-byte identical to the
// atlas-summons producer definition.
type StatusEventAttackedTarget struct {
	MonsterId uint32 `json:"monsterId"`
	Damage    uint32 `json:"damage"`
}

type StatusEventAttackedBody struct {
	Direction byte                        `json:"direction"`
	Targets   []StatusEventAttackedTarget `json:"targets"`
}
