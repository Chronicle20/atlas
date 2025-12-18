package wishlist

import (
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus        = "EVENT_TOPIC_WISHLIST_STATUS"
	StatusEventTypeAdded       = "ADDED"
	StatusEventTypeDeleted     = "DELETED"
	StatusEventTypeDeletedAll  = "DELETED_ALL"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventAddedBody struct {
	SerialNumber uint32 `json:"serialNumber"`
	ItemId       uuid.UUID `json:"itemId"`
}

type StatusEventDeletedBody struct {
	ItemId uuid.UUID `json:"itemId"`
}

type StatusEventDeletedAllBody struct {
	// Empty body as no additional information is needed for deletion of all items
}