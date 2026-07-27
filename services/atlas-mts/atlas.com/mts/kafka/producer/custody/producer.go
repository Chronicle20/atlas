package custody

import (
	"atlas-mts/kafka/message/custody"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// keyFor derives a stable partition key from a uuid (its first 4 bytes), so all
// acks for one transaction land on the same partition in order.
func keyFor(id uuid.UUID) []byte {
	return producer.CreateKey(int(binary.LittleEndian.Uint32(id[:4])))
}

// AcceptedStatusEventProvider builds an ACCEPTED ack for a created (or
// already-present) listing, echoing the transactionId and listingId.
func AcceptedStatusEventProvider(transactionId uuid.UUID, listingId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventAcceptedBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventTypeAccepted,
		Body: custody.StatusEventAcceptedBody{
			ListingId: listingId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ReleasedStatusEventProvider builds a RELEASED ack for a soft-deleted (or
// already-released) holding, echoing the transactionId and holdingId.
func ReleasedStatusEventProvider(transactionId uuid.UUID, holdingId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventReleasedBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventTypeReleased,
		Body: custody.StatusEventReleasedBody{
			HoldingId: holdingId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// RestoredStatusEventProvider builds a RESTORED ack for an un-soft-deleted (or
// already-live) holding, echoing the transactionId and holdingId.
func RestoredStatusEventProvider(transactionId uuid.UUID, holdingId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventRestoredBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventTypeRestored,
		Body: custody.StatusEventRestoredBody{
			HoldingId: holdingId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// MovedStatusEventProvider builds a MOVED ack for a settled listing whose
// custody moved to the buyer's holding (or was already moved on replay),
// echoing the transactionId, listingId, and the created holdingId.
func MovedStatusEventProvider(transactionId uuid.UUID, listingId uuid.UUID, holdingId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventMovedBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventTypeMoved,
		Body: custody.StatusEventMovedBody{
			ListingId: listingId,
			HoldingId: holdingId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ErrorStatusEventProvider builds an ERROR ack carrying the transactionId and a
// failure message.
func ErrorStatusEventProvider(transactionId uuid.UUID, errMsg string) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventErrorBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventTypeError,
		Body: custody.StatusEventErrorBody{
			Error: errMsg,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}
