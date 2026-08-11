package escrow

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	custodymsg "atlas-trades/kafka/message/custody"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Acks are keyed by the escrow row id — the same key the orchestrator's commands
// use — so every message about one row stays on one partition and cannot be
// reordered relative to the others.
func ackKey(escrowId uuid.UUID) []byte {
	return producer.CreateKey(int(escrowId.ID()))
}

func acceptedStatusProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custodymsg.StatusEvent[custodymsg.StatusEventAcceptedBody]{
		TransactionId: transactionId,
		Type:          custodymsg.StatusEventTypeAccepted,
		Body:          custodymsg.StatusEventAcceptedBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(ackKey(escrowId), value)
}

func releasedStatusProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custodymsg.StatusEvent[custodymsg.StatusEventReleasedBody]{
		TransactionId: transactionId,
		Type:          custodymsg.StatusEventTypeReleased,
		Body:          custodymsg.StatusEventReleasedBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(ackKey(escrowId), value)
}

func restoredStatusProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custodymsg.StatusEvent[custodymsg.StatusEventRestoredBody]{
		TransactionId: transactionId,
		Type:          custodymsg.StatusEventTypeRestored,
		Body:          custodymsg.StatusEventRestoredBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(ackKey(escrowId), value)
}

// removedStatusProvider acks a hard delete. The orchestrator does not route it
// into step completion — see StatusEventTypeRemoved.
func removedStatusProvider(transactionId uuid.UUID, escrowId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custodymsg.StatusEvent[custodymsg.StatusEventRemovedBody]{
		TransactionId: transactionId,
		Type:          custodymsg.StatusEventTypeRemoved,
		Body:          custodymsg.StatusEventRemovedBody{EscrowId: escrowId},
	}
	return producer.SingleMessageProvider(ackKey(escrowId), value)
}

// errorStatusProvider fails the saga step. It carries the reason so the
// orchestrator's log names the actual database failure rather than a generic
// "custody operation failed".
func errorStatusProvider(transactionId uuid.UUID, escrowId uuid.UUID, reason string) model.Provider[[]kafka.Message] {
	value := &custodymsg.StatusEvent[custodymsg.StatusEventErrorBody]{
		TransactionId: transactionId,
		Type:          custodymsg.StatusEventTypeError,
		Body:          custodymsg.StatusEventErrorBody{EscrowId: escrowId, Error: reason},
	}
	return producer.SingleMessageProvider(ackKey(escrowId), value)
}
