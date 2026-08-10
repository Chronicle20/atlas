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

// CommandProvider keys the saga command by its transaction id so every command
// for one saga lands on the same partition and is processed in order.
func CommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
