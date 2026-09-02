package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const EnvEventTopicDragonStatus topic.Token = "EVENT_TOPIC_DRAGON_STATUS"

const (
	EventDragonStatusCreated   = "CREATED"
	EventDragonStatusMoved     = "MOVED"
	EventDragonStatusDestroyed = "DESTROYED"
)

// StatusEvent is the dragon-status event envelope. It is exported because
// atlas-channel consumes it across a MODULE boundary — the channel-side mirror
// at services/atlas-channel/.../kafka/message/dragon/kafka.go must keep every
// json tag byte-for-byte identical. A tag renamed in one and not the other
// fails no build and decodes into a zero-valued body at runtime.
//
// The dragon has no id of its own; OwnerCharacterId is the identity (the client
// addresses all three clientbound ops by owner character id).
type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

// StatusEventCreatedBody carries the dragon's spawn frame. X/Y are int32 because
// SPAWN_DRAGON encodes 4-byte coordinates.
type StatusEventCreatedBody struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Stance byte   `json:"stance"`
	JobId  uint16 `json:"jobId"`
}

// StatusEventMovedBody carries the raw CMovePath blob and no coordinates: the
// blob is what other clients render. The stored position exists only so a
// late-entering viewer gets a sane first frame.
type StatusEventMovedBody struct {
	RawMovement []byte `json:"rawMovement"`
}

// StatusEventDestroyedBody is empty: REMOVE_DRAGON carries only the owner id,
// which lives on the envelope.
type StatusEventDestroyedBody struct{}
