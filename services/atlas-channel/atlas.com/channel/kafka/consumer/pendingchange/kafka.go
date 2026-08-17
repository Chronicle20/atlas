package pendingchange

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EnvEventTopic is the channel-side copy of atlas-character's pending-change
// event topic env key
// (services/atlas-character/atlas.com/character/kafka/message/pending_change/kafka.go).
const EnvEventTopic = "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"

const (
	EventTypeCreated  = "PENDING_CHANGE_CREATED"
	EventTypeResolved = "PENDING_CHANGE_RESOLVED"
)

// ChangeType values. Mirrors atlas-character pending_change.TypeNameChange /
// pending_change.TypeWorldTransfer.
const (
	ChangeTypeNameChange    = "NAME_CHANGE"
	ChangeTypeWorldTransfer = "WORLD_TRANSFER"
)

// Status values. Mirrors atlas-character pending_change.Status*.
const (
	StatusPending   = "PENDING"
	StatusApplied   = "APPLIED"
	StatusCancelled = "CANCELLED"
	StatusRejected  = "REJECTED"
	StatusExpired   = "EXPIRED"
)

// ReasonNameTaken mirrors the reject reason atlas-character emits
// (pending_change/eligibility.go, pending_change/processor.go) when a pending
// name change is invalidated because another character claimed the requested
// name first. This is the only Reason value this consumer branches on; every
// other reason string is opaque here and only reaches the player as pink
// text.
const ReasonNameTaken = "name_taken"

// StatusEvent is the channel-side copy of the atlas-character D1 pending-change
// status envelope. Field-for-field byte-identical to atlas-character's
// kafka/message/pending_change.StatusEvent so the same JSON deserializes on
// the channel side.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	WorldId       world.Id  `json:"worldId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ResolvedEventBody is the channel-side copy of
// atlas-character's pending_change.ResolvedEventBody. Field names and json
// tags must mirror the producer exactly, or the body decodes zero-valued at
// runtime with no build error.
type ResolvedEventBody struct {
	PendingChangeId    uuid.UUID `json:"pendingChangeId"`
	ChangeType         string    `json:"changeType"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	RequestedName      string    `json:"requestedName"`
	DestinationWorldId world.Id  `json:"destinationWorldId"`
}
