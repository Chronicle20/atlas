package summon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const EnvEventTopicSummonStatus = "EVENT_TOPIC_SUMMON_STATUS"

const (
	EventSummonStatusCreated   = "CREATED"
	EventSummonStatusMoved     = "MOVED"
	EventSummonStatusAttacked  = "ATTACKED"
	EventSummonStatusDamaged   = "DAMAGED"
	EventSummonStatusDestroyed = "DESTROYED"
	EventSummonStatusSkill     = "SKILL"
)

// StatusEvent is the summon-status event envelope. It is exported because
// atlas-channel consumes it across the service boundary.
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

// StatusEventDamagedBody carries the damage applied to a summon and the id of the
// monster that dealt it. Consumed by atlas-channel to rebroadcast the
// SummonDamage clientbound packet.
type StatusEventDamagedBody struct {
	Damage        int32  `json:"damage"`
	MonsterIdFrom uint32 `json:"monsterIdFrom"`
}

// StatusEventAttackedTarget is one {monster, clamped damage} pair carried by an
// ATTACKED event. The damage is the server-clamped value, not the raw client
// report.
type StatusEventAttackedTarget struct {
	MonsterId uint32 `json:"monsterId"`
	Damage    uint32 `json:"damage"`
}

type StatusEventAttackedBody struct {
	Direction byte                        `json:"direction"`
	Targets   []StatusEventAttackedTarget `json:"targets"`
}

// StatusEventSkillBody carries the Beholder aura skill-effect visual. NewStance
// is the animation stance the client plays (Cosmic uses 5 for the heal pulse and
// 6-8 for the buff pulse). Consumed by atlas-channel to rebroadcast the
// SummonSkill clientbound packet map-wide.
type StatusEventSkillBody struct {
	NewStance byte `json:"newStance"`
}
