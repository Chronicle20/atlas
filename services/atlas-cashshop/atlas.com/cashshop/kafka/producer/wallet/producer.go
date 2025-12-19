package wallet

import (
	"atlas-cashshop/kafka/message/wallet"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
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
	key := producer.CreateKey(int(accountId))
	value := &wallet.StatusEvent[wallet.StatusEventUpdatedBody]{
		AccountId: accountId,
		Type:      wallet.StatusEventTypeUpdated,
		Body: wallet.StatusEventUpdatedBody{
			Credit:  credit,
			Points:  points,
			Prepaid: prepaid,
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