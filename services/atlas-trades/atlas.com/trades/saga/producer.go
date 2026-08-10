// Package saga builds and emits the settlement saga atlas-trades submits to
// atlas-saga-orchestrator. atlas-trades NEVER enumerates concrete saga steps:
// it submits a single trade_settlement composite (design §6.3) and the
// orchestrator expands it into the release / accept / award_mesos sequence.
package saga

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// initiatedBy identifies this service on the saga record. The orchestrator
// echoes it back on status events and it is what a GM reading a stuck saga sees.
const initiatedBy = "atlas-trades"

// settlementStepId names the composite step. The orchestrator replaces it with
// the expanded steps, so it is only ever seen in a failure before expansion.
const settlementStepId = "trade_settlement"

// stageStepId names the transfer_to_trade composite one PUT_ITEM submits, and
// unwindStepId the trade_unwind composite a teardown submits. Same reasoning as
// settlementStepId: expansion replaces them.
const (
	stageStepId     = "transfer_to_trade"
	stageMesoStepId = "stage_meso"
	unwindStepId    = "trade_unwind"
)

// Build assembles the one-step trade_transaction saga for a settlement.
//
// transactionId is the settlement's identity end to end: the saga's id, the key
// the terminal-status consumer resolves the room by, and the ledger's
// idempotency key (FR-5.7). It is minted at the SETTLING transition and stored
// on the room, never re-derived.
//
// No timeout is set, so the orchestrator applies its default. A trade settles
// in a handful of Kafka round trips and has no external wait; overriding the
// default here would only encode a second, unmaintained number.
func Build(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) sharedsaga.Saga {
	return sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.TradeTransaction).
		SetInitiatedBy(initiatedBy).
		AddStep(settlementStepId, sharedsaga.Pending, sharedsaga.TradeSettlement, payload).
		Build()
}

// BuildStage assembles the one-step trade_staging saga one PUT_ITEM submits.
//
// transactionId is the ESCROW ROW ID. One staging saga owns exactly one escrow
// row, so making them the same value means a SAGA_FAILED needs no lookup table
// to find the dialog slot it has to free — the id it carries IS the handle.
//
// The saga type is TradeStaging, deliberately not TradeTransaction: a settlement
// is a two-party swap whose reverse-walk pairs releases with accepts by asset
// id, and a stage has no pair to find (see the TradeStaging doc comment).
func BuildStage(transactionId uuid.UUID, payload sharedsaga.TransferToTradePayload) sharedsaga.Saga {
	return sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.TradeStaging).
		SetInitiatedBy(initiatedBy).
		AddStep(stageStepId, sharedsaga.Pending, sharedsaga.TransferToTrade, payload).
		Build()
}

// BuildStageMeso assembles the one-step saga that moves staged meso.
//
// Meso needs no custody row of its own on the orchestrator side — it is a
// balance, not an asset — so this is a bare award_mesos rather than a composite:
// a NEGATIVE amount when the player raises what they are staking, a positive one
// when they lower it (design §5A.5). atlas-trades' own escrow meso row is what
// records the escrowed total, and it is written only once this saga completes.
//
// It runs as TradeStaging for the same reason the item stage does: the caller
// keys its pending state by this transaction id and routes the terminal status
// back to the room.
func BuildStageMeso(transactionId uuid.UUID, payload sharedsaga.AwardMesosPayload) sharedsaga.Saga {
	return sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.TradeStaging).
		SetInitiatedBy(initiatedBy).
		AddStep(stageMesoStepId, sharedsaga.Pending, sharedsaga.AwardMesos, payload).
		Build()
}

// BuildUnwind assembles the one-step trade_unwind saga a teardown submits: every
// escrowed item back to its owner, every escrowed meso refunded IN FULL.
//
// It runs as TradeTransaction rather than TradeStaging because an unwind, like a
// settlement, moves BOTH sides' custody at once; the terminal status routes by
// the durable record either way.
func BuildUnwind(transactionId uuid.UUID, payload sharedsaga.TradeUnwindPayload) sharedsaga.Saga {
	return sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.TradeTransaction).
		SetInitiatedBy(initiatedBy).
		AddStep(unwindStepId, sharedsaga.Pending, sharedsaga.TradeUnwind, payload).
		Build()
}

// CommandProvider keys the saga command by its transaction id so every command
// for one saga lands on the same partition and is processed in order.
func CommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
