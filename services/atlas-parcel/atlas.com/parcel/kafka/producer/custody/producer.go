package custody

import (
	"atlas-parcel/kafka/message/custody"
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
// already-present) parcel, echoing the transactionId and parcelId.
func AcceptedStatusEventProvider(transactionId uuid.UUID, parcelId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventAcceptedBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventAccepted,
		Body: custody.StatusEventAcceptedBody{
			ParcelId: parcelId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ReleasedStatusEventProvider builds a RELEASED ack for a released (or
// already-released) parcel, echoing the transactionId and parcelId.
func ReleasedStatusEventProvider(transactionId uuid.UUID, parcelId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventReleasedBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventReleased,
		Body: custody.StatusEventReleasedBody{
			ParcelId: parcelId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ErrorStatusEventProvider builds an ERROR ack carrying the transactionId and a
// failure message.
func ErrorStatusEventProvider(transactionId uuid.UUID, errMsg string) model.Provider[[]kafka.Message] {
	value := &custody.StatusEvent[custody.StatusEventErrorBody]{
		TransactionId: transactionId,
		Type:          custody.StatusEventError,
		Body: custody.StatusEventErrorBody{
			Error: errMsg,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}
