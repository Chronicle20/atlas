package cashshop

import "github.com/google/uuid"

const (
	EnvCommandTopicWallet     = "COMMAND_TOPIC_WALLET"
	CommandTypeAdjustCurrency = "ADJUST_CURRENCY"
)

// AdjustCurrencyCommand represents a command to adjust currency in a wallet
type AdjustCurrencyCommand struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CurrencyType  uint32    `json:"currencyType"` // 1=credit, 2=points, 3=prepaid
	Amount        int32     `json:"amount"`       // Can be negative for deduction
	Type          string    `json:"type"`
}
