package compartment

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_CASH_COMPARTMENT_STATUS"
)

const (
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
)

// StatusEvent represents a cash compartment status event
type StatusEvent[E any] struct {
	AccountId       uint32    `json:"accountId"`
	CharacterId     uint32    `json:"characterId"`
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	Type            string    `json:"type"`
	Body            E         `json:"body"`
}

type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
}

type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AssetId       uint32    `json:"assetId"`
	CashId        int64     `json:"cashId"`     // CashId for client notification correlation
	TemplateId    uint32    `json:"templateId"` // Item template ID for client notification
}
