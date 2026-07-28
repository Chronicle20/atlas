package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeDeleted       = "DELETED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventDeletedBody struct{}
