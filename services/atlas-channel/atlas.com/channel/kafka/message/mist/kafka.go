package mist

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopic   = "EVENT_TOPIC_MIST"
	EnvCommandTopic = "COMMAND_TOPIC_MIST"

	EventTypeCreated   = "MIST_CREATED"
	EventTypeDestroyed = "MIST_DESTROYED"

	CommandTypeCreate = "CREATE"

	// TargetKind / EffectKind mirror atlas-maps' descriptors. Empty means
	// CHARACTER / DISEASE there; the channel always sets both explicitly.
	TargetKindCharacter = "CHARACTER"
	TargetKindMonster   = "MONSTER"

	EffectKindDisease        = "DISEASE"
	EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
)

// Event is the channel-side envelope for mist events emitted by atlas-maps.
type Event[E any] struct {
	Tenant    uuid.UUID  `json:"tenant"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MistId    uuid.UUID  `json:"mistId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// Command is the envelope for mist commands published to EnvCommandTopic.
// Mirrors atlas-maps' kafka/message/mist Command.
type Command[E any] struct {
	Tenant uuid.UUID `json:"tenant"`
	Type   string    `json:"type"`
	Body   E         `json:"body"`
}

// CreateCommandBody requests creation of a new mist on the named field.
// Mirrors atlas-maps' CreateCommandBody exactly -- atlas-maps owns this
// contract. The Disease* fields are the generic status name / magnitude /
// per-target duration triple; the names are historical.
//
// DiseaseDuration, Duration, and TickIntervalMs are all MILLISECONDS.
type CreateCommandBody struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerType        string     `json:"ownerType"`
	OwnerId          uint32     `json:"ownerId"`
	OriginX          int16      `json:"originX"`
	OriginY          int16      `json:"originY"`
	LtX              int16      `json:"ltX"`
	LtY              int16      `json:"ltY"`
	RbX              int16      `json:"rbX"`
	RbY              int16      `json:"rbY"`
	Disease          string     `json:"disease"`
	DiseaseValue     int32      `json:"diseaseValue"`
	DiseaseDuration  int64      `json:"diseaseDuration"`
	Duration         int64      `json:"duration"`
	TickIntervalMs   int64      `json:"tickIntervalMs"`
	SourceSkillId    uint32     `json:"sourceSkillId"`
	SourceSkillLevel uint32     `json:"sourceSkillLevel"`
	// TargetKind is "CHARACTER" or "MONSTER"; empty means CHARACTER.
	TargetKind string `json:"targetKind"`
	// EffectKind is "DISEASE" or "DAMAGE_OVER_TIME"; empty means DISEASE.
	EffectKind string `json:"effectKind"`
}

// CreatedBody mirrors atlas-maps' MIST_CREATED payload.
type CreatedBody struct {
	OwnerType        string `json:"ownerType"`
	OwnerId          uint32 `json:"ownerId"`
	SourceSkillId    uint32 `json:"sourceSkillId"`
	SourceSkillLevel uint32 `json:"sourceSkillLevel"`
	Type             int32  `json:"type"`
	OriginX          int16  `json:"originX"`
	OriginY          int16  `json:"originY"`
	LtX              int16  `json:"ltX"`
	LtY              int16  `json:"ltY"`
	RbX              int16  `json:"rbX"`
	RbY              int16  `json:"rbY"`
	Duration         int64  `json:"duration"`
	// ElemAttr is the client's `nElemAttr`; SkillDelay is its `skillDelay`
	// draw delay (units of 100 ms). The existing `Type` field IS the client's
	// `nType` -- do not add a second key for it.
	ElemAttr   int32 `json:"elemAttr"`
	SkillDelay int16 `json:"skillDelay"`
}

// DestroyedBody mirrors atlas-maps' MIST_DESTROYED payload.
type DestroyedBody struct {
	Reason string `json:"reason"`
}
