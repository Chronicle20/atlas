package wallet

import (
	"atlas-cashshop/kafka/message/wallet"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CreateStatusEventProvider(accountId uint32, credit uint32, points uint32, prepaid uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &wallet.StatusEvent[wallet.StatusEventCreatedBody]{
		AccountId: accountId,
		Type:      wallet.StatusEventTypeCreated,
		Body: wallet.StatusEventCreatedBody{
			Credit:  credit,
			Points:  points,
			Prepaid: prepaid,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdateStatusEventProvider(accountId uint32, credit uint32, points uint32, prepaid uint32) model.Provider[[]kafka.Message] {
	return UpdateStatusEventWithTransactionProvider(accountId, credit, points, prepaid, uuid.Nil)
}

func UpdateStatusEventWithTransactionProvider(accountId uint32, credit uint32, points uint32, prepaid uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &wallet.StatusEvent[wallet.StatusEventUpdatedBody]{
		AccountId: accountId,
		Type:      wallet.StatusEventTypeUpdated,
		Body: wallet.StatusEventUpdatedBody{
			Credit:        credit,
			Points:        points,
			Prepaid:       prepaid,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ErrorStatusEventProvider reports a failed transactional wallet adjust, keyed by
// accountId (mirrors the update/create providers) and echoing the transaction id
// so the orchestrator can fail the waiting saga step fast.
func ErrorStatusEventProvider(accountId uint32, transactionId uuid.UUID, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &wallet.StatusEvent[wallet.StatusEventErrorBody]{
		AccountId: accountId,
		Type:      wallet.StatusEventTypeError,
		Body: wallet.StatusEventErrorBody{
			TransactionId: transactionId,
			Reason:        reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeleteStatusEventProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &wallet.StatusEvent[wallet.StatusEventDeletedBody]{
		AccountId: accountId,
		Type:      wallet.StatusEventTypeDeleted,
		Body:      wallet.StatusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
