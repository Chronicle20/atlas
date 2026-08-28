package wallet

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus   topic.Token = "EVENT_TOPIC_WALLET_STATUS"
	EnvCommandTopicWallet topic.Token = "COMMAND_TOPIC_WALLET"
)

const (
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"
	// StatusEventTypeError reports a failed transactional wallet adjust (missing
	// wallet, insufficient balance) so a saga waiting on the ADJUST_CURRENCY command
	// fails fast instead of waiting out its timeout. Only emitted when the command
	// carried a (non-nil) transaction id.
	StatusEventTypeError = "ERROR"

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
	Credit        uint32    `json:"credit"`
	Points        uint32    `json:"points"`
	Prepaid       uint32    `json:"prepaid"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	// SceneRefreshOwned marks that the wallet movement's originating operation
	// (e.g. a cash-shop gift) has its own status handler that announces the
	// scene refresh (CashQueryResult) in the correct client-required order.
	// A CashSceneCashShop consumer must skip its own refresh when this is set,
	// so the two do not race across topics. Every other producer -- MTS settle,
	// GM @award, arbitrary saga AdjustCurrency ingress -- leaves this unset and
	// keeps its existing refresh with zero behavior change.
	SceneRefreshOwned bool `json:"sceneRefreshOwned,omitempty"`
}

type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}

// StatusEventErrorBody reports a failed transactional wallet adjust. TransactionId
// echoes the ADJUST_CURRENCY command so the orchestrator can fail the matching
// saga step; Reason is a human-readable diagnostic (not client-facing).
type StatusEventErrorBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Reason        string    `json:"reason"`
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
