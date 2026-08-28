// Package monsterstatus mirrors the atlas-monsters monster status events this
// service CONSUMES (source of truth:
// services/atlas-monsters/atlas.com/monsters/monster/kafka.go). Only the
// consumed envelope fields and status types are mirrored — Task 26 needs
// UniqueId/MonsterId/Type and the provenance echo, never the per-type Body,
// so Body stays generic (json.RawMessage at the call site). Unknown status
// types on the topic are ignored by the handler's type guard.
package monsterstatus

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicMonsterStatus topic.Token = "EVENT_TOPIC_MONSTER_STATUS"
)

const (

	// EventMonsterStatusCreated/Killed/Destroyed are the only status types
	// FR-B18's elimination tracking consumes (monsters.go).
	EventMonsterStatusCreated   = "CREATED"
	EventMonsterStatusKilled    = "KILLED"
	EventMonsterStatusDestroyed = "DESTROYED"
)

// StatusEvent is the envelope every monster status event arrives in.
// SpawnSourceType/SpawnSourceId echo the monster's provenance (FR-P3): a
// monster belongs to a CRIMSON_BALROG occurrence only if SpawnSourceType is
// "EVENT" and SpawnSourceId is that occurrence's id.
type StatusEvent[E any] struct {
	UniqueId        uint32 `json:"uniqueId"`
	MonsterId       uint32 `json:"monsterId"`
	Type            string `json:"type"`
	SpawnSourceType string `json:"spawnSourceType,omitempty"`
	SpawnSourceId   string `json:"spawnSourceId,omitempty"`
	Body            E      `json:"body"`
}
