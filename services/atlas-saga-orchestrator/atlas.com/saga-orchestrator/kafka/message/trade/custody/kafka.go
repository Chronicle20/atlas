// This is the orchestrator's own copy of the atlas-trades custody wire
// contract. The orchestrator cannot import the atlas-trades module, so these
// structs mirror
// services/atlas-trades/atlas.com/trades/kafka/message/custody/kafka.go
// byte-for-byte (identical JSON tags + Type discriminator strings). This
// follows the mts/custody and cashshop/compartment precedent.
//
// Trade escrow exists because a staged item must genuinely LEAVE its owner's
// compartment (task-205 design §5A): the resulting inventory delta is what
// clears the client's m_bExclRequestSent, and nothing else in the trade flow
// does. The escrow row is the durable custody record that makes returning the
// item possible without knowing which saga staged it.
package custody

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	// EnvCommandTopic is the env var naming the trade custody command topic.
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_TRADE_CUSTODY"
)

const (
	CommandAcceptToTrade    = "ACCEPT_TO_TRADE"
	CommandReleaseFromTrade = "RELEASE_FROM_TRADE"
	// CommandRestoreTradeEscrow un-soft-deletes an escrow row (the late-comp
	// inverse of ReleaseFromTrade).
	CommandRestoreTradeEscrow = "RESTORE_TRADE_ESCROW"
	// CommandRemoveTradeEscrow hard-deletes a spurious escrow row (the
	// late-comp inverse of AcceptToTrade).
	CommandRemoveTradeEscrow = "REMOVE_TRADE_ESCROW"
)

// Command is the generic custody command envelope. TransactionId keys the saga
// step; Type discriminates which body is carried.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// AcceptToTradeCommandBody carries every field needed to CREATE an escrow row.
//
// The item state is the SHARED sharedsaga.AssetSnapshot rather than a stat list
// spelled out here. The spelled-out list this replaced omitted Expiration,
// CashId, Rechargeable, LevelType, Experience, HammersApplied and the whole pet
// block, which meant a cash item, a pet or a timed item came back out of escrow
// degraded — stripped of its cash serial, its expiry and its pet identity. Cash
// items and pets are stageable (trade/restriction.go blocks only equipped items,
// the untradeable flags and the WZ tradeBlock), so that was reachable.
//
// Nesting the snapshot does not compromise the row's COPY/restore column-order
// discipline: escrow.toItemEntity still explodes it into explicit name-keyed
// columns, exactly as atlas-mts's holdings table does.
//
// Snapshot.Slot is the source slot the item came from, Snapshot.TemplateId and
// Snapshot.Quantity the staged template and STAGED amount (a partial stage of
// 1-of-40 escrows 1).
type AcceptToTradeCommandBody struct {
	EscrowId            uuid.UUID `json:"escrowId"`
	RoomId              uuid.UUID `json:"roomId"`
	OwnerId             uint32    `json:"ownerId"`
	TradeSlot           byte      `json:"tradeSlot"`
	SourceInventoryType byte      `json:"sourceInventoryType"`
	AssetId             uint32    `json:"assetId"`

	Snapshot sharedsaga.AssetSnapshot `json:"snapshot"`
}

// ReleaseFromTradeCommandBody soft-deletes the escrow row. The row holds the
// snapshot, so a release can never disagree with the accept that created it.
type ReleaseFromTradeCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// RestoreTradeEscrowCommandBody un-soft-deletes the escrow row by id.
type RestoreTradeEscrowCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// RemoveTradeEscrowCommandBody hard-deletes a spurious escrow row by id.
type RemoveTradeEscrowCommandBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

const (
	// EnvStatusEventTopic names the trade custody status (ack) topic.
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_TRADE_CUSTODY_STATUS"
)

const (
	StatusEventTypeAccepted = "ACCEPTED"
	StatusEventTypeReleased = "RELEASED"
	StatusEventTypeRestored = "RESTORED"
	// StatusEventTypeRemoved acks a HARD delete. It is deliberately distinct
	// from RELEASED: both a restore and a remove are LATE compensating inverses
	// dispatched after their saga already terminated, so neither may be routed
	// into StepCompleted — doing so would advance whatever step happened to be
	// current. Reusing RELEASED here would have made that depend on the saga
	// already being evicted from the cache, which is timing, not a contract.
	StatusEventTypeRemoved = "REMOVED"
	StatusEventTypeError   = "ERROR"
)

// StatusEvent is the generic custody ack envelope. TransactionId echoes the
// command so the orchestrator can complete/fail the saga step.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventAcceptedBody acks escrow creation, echoing the escrow id.
type StatusEventAcceptedBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventReleasedBody acks an escrow release, echoing the escrow id.
type StatusEventReleasedBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventRestoredBody acks an escrow restore, echoing the escrow id.
type StatusEventRestoredBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventRemovedBody acks an escrow hard delete, echoing the escrow id.
type StatusEventRemovedBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
}

// StatusEventErrorBody reports a custody failure with a human-readable reason.
type StatusEventErrorBody struct {
	EscrowId uuid.UUID `json:"escrowId"`
	Error    string    `json:"error"`
}
