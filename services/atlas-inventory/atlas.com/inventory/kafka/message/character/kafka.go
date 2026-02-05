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

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Name string `json:"name"`
}

type DeletedStatusEventBody struct {
}
