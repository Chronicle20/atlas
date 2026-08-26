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
	// SceneRefreshOwned marks that the wallet movement's originating operation
	// (e.g. a cash-shop gift) has its own status handler that announces the
	// scene refresh (CashQueryResult) in the correct client-required order.
	// A CashSceneCashShop consumer must skip its own refresh when this is set,
	// so the two do not race across topics. Every other producer -- MTS settle,
	// GM @award, arbitrary saga AdjustCurrency ingress -- leaves this unset and
	// keeps its existing refresh with zero behavior change.
	SceneRefreshOwned bool `json:"sceneRefreshOwned,omitempty"`
}
