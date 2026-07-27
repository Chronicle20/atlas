package saga

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Re-export the types and constants atlas-mts needs from the shared atlas-saga
// library. Mirrors the character-factory saga package: the service constructs
// sagas against these local aliases and emits them to COMMAND_TOPIC_SAGA.
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action
	Step   = sharedsaga.Step[any]

	// Payload types used by the list flow.
	AwardMesosPayload    = sharedsaga.AwardMesosPayload
	TransferToMtsPayload = sharedsaga.TransferToMtsPayload

	// Payload type used by the take-home flow.
	WithdrawFromMtsPayload = sharedsaga.WithdrawFromMtsPayload

	// Payload type used by the buy/buy-now settlement flow. The orchestrator
	// expands MtsSettlePurchase into award_currency(buyer prepaid -markedUp) +
	// award_currency(seller points +listValue) + mts_move_listing_to_holding.
	MtsSettlePurchasePayload = sharedsaga.MtsSettlePurchasePayload

	// Payload types used by the auction bidding flow. MtsBidEscrow is a single-step
	// wallet adjust the orchestrator routes straight to the cash-shop wallet
	// (negative to HOLD the bidder's prepaid at bid time, positive to RELEASE an
	// outbid bidder's escrow). AwardCurrency + MtsMoveListingToHolding compose the
	// auction-settle (seller points credit + custody move to the winner) WITHOUT a
	// buyer-debit step — the winner was already debited at bid time, so reusing
	// MtsSettlePurchase here would double-debit the winner.
	MtsBidEscrowPayload            = sharedsaga.MtsBidEscrowPayload
	AwardCurrencyPayload           = sharedsaga.AwardCurrencyPayload
	MtsMoveListingToHoldingPayload = sharedsaga.MtsMoveListingToHoldingPayload
)

const (
	// Saga types
	MtsOperation = sharedsaga.MtsOperation

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardMesos              = sharedsaga.AwardMesos
	TransferToMts           = sharedsaga.TransferToMts
	WithdrawFromMts         = sharedsaga.WithdrawFromMts
	MtsSettlePurchase       = sharedsaga.MtsSettlePurchase
	MtsBidEscrow            = sharedsaga.MtsBidEscrow
	AwardCurrency           = sharedsaga.AwardCurrency
	MtsMoveListingToHolding = sharedsaga.MtsMoveListingToHolding
)

// Currency type constants for AwardCurrency payloads. They mirror the cash-shop
// wallet bucket numbering the orchestrator uses (2=points seller credit, 3=prepaid
// bidder escrow).
const (
	CurrencyTypePoints  = uint32(2)
	CurrencyTypePrepaid = uint32(3)
)
