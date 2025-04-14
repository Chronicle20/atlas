package asset

import (
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus            = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeMoved           = "MOVED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	TemplateId    uint32    `json:"templateId"`
	Slot          int16     `json:"slot"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

type DeletedStatusEventBody struct {
}

type MovedStatusEventBody struct {
	OldSlot int16 `json:"oldSlot"`
}

type QuantityChangedEventBody struct {
	Quantity uint32 `json:"quantity"`
}
