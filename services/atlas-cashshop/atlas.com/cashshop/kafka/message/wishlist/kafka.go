package wishlist

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_WISHLIST_STATUS"
)

const (
	StatusEventTypeAdded      = "ADDED"
	StatusEventTypeDeleted    = "DELETED"
	StatusEventTypeDeletedAll = "DELETED_ALL"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventAddedBody struct {
	SerialNumber uint32    `json:"serialNumber"`
	ItemId       uuid.UUID `json:"itemId"`
}

type StatusEventDeletedBody struct {
	ItemId uuid.UUID `json:"itemId"`
}

type StatusEventDeletedAllBody struct {
	// Empty body as no additional information is needed for deletion of all items
}
