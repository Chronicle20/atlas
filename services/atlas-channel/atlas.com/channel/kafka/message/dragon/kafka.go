package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// This file MIRRORS the atlas-dragons contract across a module boundary.
// Command direction: channel -> dragons (producer here, consumer there).
// Event direction: dragons -> channel (producer there, consumer here).
//
// Authoritative definitions:
//   services/atlas-dragons/atlas.com/dragons/kafka/consumer/dragon/kafka.go
//   services/atlas-dragons/atlas.com/dragons/dragon/kafka.go
//
// Every json tag MUST stay byte-for-byte identical. The two files are in
// separate Go modules, so a tag changed in one and not the other fails no build
// — it decodes into a zero-valued body at runtime, silently. kafka_test.go
// pins the wire shape from literal JSON so that divergence fails a test instead.

const EnvCommandTopic topic.Token = "COMMAND_TOPIC_DRAGON"

const (
	CommandTypeCreate  = "CREATE"
	CommandTypeDestroy = "DESTROY"
	CommandTypeMove    = "MOVE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreateCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type DestroyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type MoveCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	Stance      byte   `json:"stance"`
	RawMovement []byte `json:"rawMovement"`
}

const EnvEventTopicDragonStatus topic.Token = "EVENT_TOPIC_DRAGON_STATUS"

const (
	EventDragonStatusCreated   = "CREATED"
	EventDragonStatusMoved     = "MOVED"
	EventDragonStatusDestroyed = "DESTROYED"
)

type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

// X/Y are int32: SPAWN_DRAGON encodes 4-byte coordinates.
type StatusEventCreatedBody struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Stance byte   `json:"stance"`
	JobId  uint16 `json:"jobId"`
}

type StatusEventMovedBody struct {
	RawMovement []byte `json:"rawMovement"`
}

type StatusEventDestroyedBody struct{}
