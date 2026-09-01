package asset

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_ASSET_STATUS"
)

const (
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	TemplateId    uint32    `json:"templateId"`
	Slot          int16     `json:"slot"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Quantity uint32 `json:"quantity"`
}

type DeletedStatusEventBody struct{}

type QuantityChangedStatusEventBody struct {
	Quantity uint32 `json:"quantity"`
}
