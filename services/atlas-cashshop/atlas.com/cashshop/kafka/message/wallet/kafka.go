package wallet

import "github.com/google/uuid"

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_WALLET_STATUS"
	EnvCommandTopicWallet  = "COMMAND_TOPIC_WALLET"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"

	CommandTypeAdjustCurrency = "ADJUST_CURRENCY"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type StatusEventCreatedBody struct {
	Credit  uint32 `json:"credit"`
	Points  uint32 `json:"points"`
	Prepaid uint32 `json:"prepaid"`
}

type StatusEventUpdatedBody struct {
	Credit  uint32 `json:"credit"`
	Points  uint32 `json:"points"`
	Prepaid uint32 `json:"prepaid"`
}

type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}

// Command represents a command to modify wallet state
type Command struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	Type          string    `json:"type"`
}

// AdjustCurrencyCommand represents a command to adjust currency in a wallet
type AdjustCurrencyCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CurrencyType  uint32    `json:"currencyType"` // 1=credit, 2=points, 3=prepaid
	Amount        int32     `json:"amount"`       // Can be negative for deduction
	Type          string    `json:"type"`
}