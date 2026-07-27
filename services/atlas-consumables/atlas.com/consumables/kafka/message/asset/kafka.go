package asset

import "github.com/google/uuid"

const (
	EnvEventTopicStatus            = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
)

// StatusEvent mirrors the asset status envelope emitted by atlas-inventory
// (services/atlas-inventory/atlas.com/inventory/kafka/message/asset/kafka.go).
// The reward flow subscribes to this topic solely to observe the CREATED
// confirmation atlas-inventory emits when a CREATE_ASSET succeeds — that is
// the success signal for the reward grant. It is emitted here, NOT on the
// compartment status topic (which only emits CREATED for compartment creation),
// so the reward success once-handler must key off this event.
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

// CreatedStatusEventBody is the opaque body of an asset CREATED event. The
// reward flow correlates on the envelope's transactionId, not on any body
// field, so no fields are decoded here.
type CreatedStatusEventBody struct{}
