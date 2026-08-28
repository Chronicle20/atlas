package pending_change

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopic topic.Token = "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"

	EventTypeCreated  = "PENDING_CHANGE_CREATED"
	EventTypeResolved = "PENDING_CHANGE_RESOLVED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	WorldId       world.Id  `json:"worldId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedEventBody struct {
	PendingChangeId    uuid.UUID `json:"pendingChangeId"`
	ChangeType         string    `json:"changeType"`
	RequestedName      string    `json:"requestedName"`
	DestinationWorldId world.Id  `json:"destinationWorldId"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

type ResolvedEventBody struct {
	PendingChangeId    uuid.UUID `json:"pendingChangeId"`
	ChangeType         string    `json:"changeType"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	RequestedName      string    `json:"requestedName"`
	DestinationWorldId world.Id  `json:"destinationWorldId"`
}
