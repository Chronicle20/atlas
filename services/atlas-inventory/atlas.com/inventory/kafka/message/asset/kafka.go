package asset

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
	StatusEventTypeMoved   = "MOVED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

type DeletedStatusEventBody struct {
}

type MovedStatusEventBody struct {
	NewSlot int16 `json:"newSlot"`
	OldSlot int16 `json:"oldSlot"`
}
