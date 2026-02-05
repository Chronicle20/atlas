package character

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

// StatusEvent represents a character status event
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// CreatedStatusBody represents the body of a character created event
type CreatedStatusBody struct {
	Name string `json:"name"`
}

// DeletedStatusEventBody represents the body of a character deleted event
type DeletedStatusEventBody struct {
}
