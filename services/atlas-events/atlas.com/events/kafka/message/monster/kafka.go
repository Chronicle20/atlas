// Package monster mirrors the atlas-monsters field commands this service
// PRODUCES (source of truth:
// services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go).
// Only the produced types/fields are mirrored, and — unlike a consumer
// mirror — they are exported: this package is the producer side of the
// contract, so events/crimsonbalrog needs to construct these values from
// another package.
//
// FieldCommand carries no monsterId at the envelope level (unlike the
// per-monster-instance `command[E]` wrapper in the source of truth); it is
// the field-scoped variant used by SPAWN_FIELD and DESTROY_BY_SOURCE.
package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_MONSTER"
)

const (
	CommandTypeSpawnField      = "SPAWN_FIELD"
	CommandTypeDestroyBySource = "DESTROY_BY_SOURCE"
)

// FieldCommand is the envelope every field-scoped monster command rides in.
type FieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// SpawnFieldCommandBody asks atlas-monsters to spawn one monster at a
// position, optionally carrying provenance (FR-P1/FR-B22). SpawnSourceType
// "EVENT" plus SpawnSourceId (the occurrence id) is what later lets
// DESTROY_BY_SOURCE clean up every monster this occurrence spawned.
type SpawnFieldCommandBody struct {
	MonsterId       uint32 `json:"monsterId"`
	X               int16  `json:"x"`
	Y               int16  `json:"y"`
	Fh              int16  `json:"fh"`
	Team            int8   `json:"team"`
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
}

// DestroyBySourceCommandBody despawns every monster in the field matching the
// provenance pair. Used by Task 27's Complete/cleanup path.
type DestroyBySourceCommandBody struct {
	SpawnSourceType string `json:"spawnSourceType"`
	SpawnSourceId   string `json:"spawnSourceId"`
}
