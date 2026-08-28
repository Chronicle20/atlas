package kite

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_KITE"
)

const (
	CommandKiteCreate  = "CREATE"
	CommandKiteDestroy = "DESTROY"
)

// Command is produced by atlas-channel and keyed on characterId, so one
// character's placements are totally ordered within a partition. That ordering
// is what makes the one-kite-per-character invariant safe without a lock; the
// per-map cap is NOT covered by it (two characters land on two partitions) and
// takes the per-field lock in the processor instead.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// CreateCommandBody carries the owner's name and position from atlas-channel.
// Both are server-side state: the client never sends coordinates for a kite
// (the sub-body is the message alone) and the name is read from the character
// record, never from the packet.
type CreateCommandBody struct {
	Name       string `json:"name"`
	TemplateId uint32 `json:"templateId"`
	Message    string `json:"message"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type DestroyCommandBody struct {
	KiteId uint32 `json:"kiteId"`
}

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_KITE_STATUS"
)

const (
	EventTopicStatusTypeCreated        = "CREATED"
	EventTopicStatusTypeDestroyed      = "DESTROYED"
	EventTopicStatusTypeCreationFailed = "CREATION_FAILED"
)

// Destroy reasons.
const (
	DestroyReasonOwnerLeft      = "OWNER_LEFT"
	DestroyReasonOwnerLoggedOut = "OWNER_LOGGED_OUT"
)

// Creation-failure reasons. FieldKiteError has an EMPTY body, so the client
// only ever renders a generic failure — these values exist so the refusal is
// diagnosable in logs.
const (
	FailureReasonMapFull        = "MAP_FULL"
	FailureReasonAlreadyPlaced  = "ALREADY_PLACED"
	FailureReasonMapForbidden   = "MAP_FORBIDDEN"
	FailureReasonMessageTooLong = "MESSAGE_TOO_LONG"
)

// StatusEvent is produced by atlas-kites and keyed on mapId for per-map
// ordering, matching the chalkboard/mist producers.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type CreatedStatusEventBody struct {
	KiteId     uint32 `json:"kiteId"`
	Name       string `json:"name"`
	TemplateId uint32 `json:"templateId"`
	Message    string `json:"message"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type DestroyedStatusEventBody struct {
	KiteId uint32 `json:"kiteId"`
	Reason string `json:"reason"`
}

// CreationFailedStatusEventBody targets a single character: CANNOT_SPAWN_KITE
// goes to the requester only, never to the map. The envelope's CharacterId is
// the addressee.
type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}
