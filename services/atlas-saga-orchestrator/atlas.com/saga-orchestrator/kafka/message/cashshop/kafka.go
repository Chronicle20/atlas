package cashshop

import "github.com/google/uuid"

const (
	EnvCommandTopicWallet     = "COMMAND_TOPIC_WALLET"
	CommandTypeAdjustCurrency = "ADJUST_CURRENCY"

	// Wallet status event constants
	EnvEventTopicWalletStatus = "EVENT_TOPIC_WALLET_STATUS"
	StatusEventTypeCreated    = "CREATED"
	StatusEventTypeUpdated    = "UPDATED"
	StatusEventTypeDeleted    = "DELETED"
	// StatusEventTypeError reports a failed transactional wallet adjust, so an
	// AwardCurrency/MtsBidEscrow saga step fails fast instead of timing out.
	StatusEventTypeError = "ERROR"
)

// AdjustCurrencyCommand represents a command to adjust currency in a wallet
type AdjustCurrencyCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CurrencyType  uint32    `json:"currencyType"` // 1=credit, 2=points, 3=prepaid
	Amount        int32     `json:"amount"`       // Can be negative for deduction
	Type          string    `json:"type"`
}

// StatusEvent represents a wallet status event from atlas-cashshop
type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// StatusEventUpdatedBody represents the body of a wallet updated event
type StatusEventUpdatedBody struct {
	Credit        uint32    `json:"credit"`
	Points        uint32    `json:"points"`
	Prepaid       uint32    `json:"prepaid"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

// StatusEventErrorBody represents the body of a failed transactional wallet
// adjust; TransactionId echoes the command so the saga step can be failed.
type StatusEventErrorBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Reason        string    `json:"reason"`
}
