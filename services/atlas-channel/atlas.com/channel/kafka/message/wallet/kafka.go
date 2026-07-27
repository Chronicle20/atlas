package wallet

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_WALLET_STATUS"
	StatusEventTypeUpdated = "UPDATED"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type StatusEventUpdatedBody struct {
	Credit        uint32    `json:"credit"`
	Points        uint32    `json:"points"`
	Prepaid       uint32    `json:"prepaid"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}
