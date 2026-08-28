// Package character carries atlas-mts's OWN copy of the character status event
// contract it consumes. There is no shared module linking atlas-mts to
// atlas-character's producer, so every type here must be diffed field-for-field
// against the producer's definitions
// (services/atlas-character/atlas.com/character/kafka/message/character/kafka.go)
// on every change — a mismatched json tag compiles cleanly and silently drops
// the event.
package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	// EnvStatusEventTopic names the character status/event topic atlas-character
	// publishes to. Mirrors that producer's EnvEventTopicCharacterStatus
	// ("EVENT_TOPIC_CHARACTER_STATUS") — this service's own message packages name
	// the const EnvStatusEventTopic (see kafka/message/mts, kafka/message/custody).
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"

	// StatusEventTypeNameChanged is the only status type atlas-mts currently
	// consumes from this topic: it keeps listing.seller_name current for display.
	StatusEventTypeNameChanged = "NAME_CHANGED"
)

// StatusEvent is the generic character status/event envelope, mirrored
// field-for-field (name, order, json tag, type) against the producer's
// StatusEvent[E any] at
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:265.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventNameChangedBody is mirrored field-for-field against the producer's
// StatusEventNameChangedBody at
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:359.
type StatusEventNameChangedBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}
