package mist

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_MIST"
	EnvEventTopic   = "EVENT_TOPIC_MIST"

	CommandTypeCreate = "CREATE"
	CommandTypeCancel = "CANCEL"

	EventTypeCreated   = "MIST_CREATED"
	EventTypeDestroyed = "MIST_DESTROYED"

	ReasonExpired   = "EXPIRED"
	ReasonCancelled = "CANCELLED"
)

// Command is the envelope for mist commands published to EnvCommandTopic.
type Command[E any] struct {
	Tenant uuid.UUID `json:"tenant"`
	Type   string    `json:"type"`
	Body   E         `json:"body"`
}

// CreateCommandBody requests creation of a new mist on the named field.
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
}

// CancelCommandBody requests cancellation of an existing mist by id.
type CancelCommandBody struct {
	MistId uuid.UUID `json:"mistId"`
}

// Event is the envelope for mist events published to EnvEventTopic.
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

// CreatedBody describes a mist that was just created.
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
}

// DestroyedBody describes a mist that was just destroyed.
type DestroyedBody struct {
	Reason string `json:"reason"`
}
