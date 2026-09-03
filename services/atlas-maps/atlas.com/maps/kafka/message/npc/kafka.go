package npc

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicNpcStatus topic.Token = "EVENT_TOPIC_NPC_STATUS"
)

const (
	EventNpcStatusTypeCreated = "CREATED"
)

// StatusEvent is the envelope for scripted-NPC lifecycle events published to
// EnvEventTopicNpcStatus. It mirrors the sibling monster package's
// StatusEvent[E] shape (services/atlas-maps/atlas.com/maps/kafka/message/monster/kafka.go)
// -- keyed by UniqueId, no TransactionId -- rather than map/weather's
// per-map StatusEvent, because a scripted-NPC placement is per-object, not
// per-map, and map/npc.Processor.Create has no transaction id to carry
// (task-290's spawn_npc saga action does not thread one through).
type StatusEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// CreatedStatusEventBody carries the placement of a newly spawned scripted
// NPC -- exactly the fields map/npc.Model holds (NpcId, X, Y, Fh). Cosmic's
// spawnNpc also sets cy, facing, and rx0/rx1 walk-range bounds on the live
// NPC object (AbstractPlayerInteraction.java:962-973 per
// docs/tasks/task-290-cosmic-map-action-parity/context.md:213), but the
// spawn_npc saga action (task-290 C14) never records them -- there is
// nothing here to carry. See the consumer-side comment in atlas-channel
// (kafka/consumer/map/npc.go) for the values substituted at write time.
type CreatedStatusEventBody struct {
	NpcId uint32 `json:"npcId"`
	X     int16  `json:"x"`
	Y     int16  `json:"y"`
	Fh    int16  `json:"fh"`
}
