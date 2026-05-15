package inventory

import "github.com/google/uuid"

const (
	EnvEventTopicStatus           = "EVENT_TOPIC_INVENTORY_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"
	StatusEventTypeDeleted        = "DELETED"
)

// StatusEvent is the on-wire shape of an inventory status event. TransactionId
// is added with omitempty so existing consumers (atlas-cashshop) continue to
// decode the same struct; non-saga emitters serialise without the field.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

// CreationFailedStatusEventBody carries the free-form error message for
// telemetry. The orchestrator does not inspect Reason; it only flips the
// step to Failed.
type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}

type DeletedStatusEventBody struct {
}
