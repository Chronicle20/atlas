package compartment

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_CASH_COMPARTMENT"
	EnvEventTopicStatus     = "EVENT_TOPIC_CASH_COMPARTMENT_STATUS"
	CommandAccept           = "ACCEPT"
	CommandRelease          = "RELEASE"
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeError    = "ERROR"
)

// Command represents a cash shop compartment command (ACCEPT/RELEASE)
type Command[E any] struct {
	AccountId       uint32 `json:"accountId"`
	CharacterId     uint32 `json:"characterId"`
	CompartmentType byte   `json:"compartmentType"`
	Type            string `json:"type"`
	Body            E      `json:"body"`
}

// AcceptCommandBody contains the data for an ACCEPT command
type AcceptCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	CashId        int64     `json:"cashId"`
	TemplateId    uint32    `json:"templateId"`
	Quantity      uint32    `json:"quantity"`
	CommodityId   uint32    `json:"commodityId"`
	PurchasedBy   uint32    `json:"purchasedBy"`
	Flag          uint16    `json:"flag"`
}

// ReleaseCommandBody contains the data for a RELEASE command
type ReleaseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	CashId        int64     `json:"cashId"`     // CashId for client notification correlation
	TemplateId    uint32    `json:"templateId"` // Item template ID for client notification
}

// StatusEvent represents a cash compartment status event
type StatusEvent[E any] struct {
	CompartmentId   uuid.UUID `json:"compartmentId"`
	CompartmentType byte      `json:"compartmentType"`
	Type            string    `json:"type"`
	Body            E         `json:"body"`
}

// StatusEventAcceptedBody contains the data for an ACCEPTED event
type StatusEventAcceptedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

// StatusEventReleasedBody contains the data for a RELEASED event
type StatusEventReleasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}

// StatusEventErrorBody contains the data for an ERROR event
type StatusEventErrorBody struct {
	ErrorCode     string    `json:"errorCode"`
	TransactionId uuid.UUID `json:"transactionId"`
}
