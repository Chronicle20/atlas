// Package mist defines the wire shapes for atlas-maps' mist commands. The
// types mirror services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go
// byte-for-byte (matching JSON tags) so atlas-monsters can publish
// MIST_CREATE / MIST_CANCEL commands without importing across service
// boundaries.
package mist

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_MIST"

	CommandTypeCreate = "CREATE"
	CommandTypeCancel = "CANCEL"
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
