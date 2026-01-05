package asset

import (
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated = "CREATED"
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
	ReferenceId   uint32 `json:"referenceId"`
	ReferenceType string `json:"referenceType"`
	Quantity      uint32 `json:"quantity,omitempty"`
}
