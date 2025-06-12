package inventory

import (
	inventory2 "atlas-cashshop/kafka/message/cashshop/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// CreateStatusEventProvider creates a provider for inventory creation events
// According to the requirements, it should always include accountId and have an empty body
func CreateStatusEventProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &inventory2.StatusEvent[inventory2.StatusEventCreatedBody]{
		AccountId: accountId,
		Type:      inventory2.StatusEventTypeCreated,
		Body:      inventory2.StatusEventCreatedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// DeleteStatusEventProvider creates a provider for inventory deletion events
// According to the requirements, it should always include accountId and have an empty body
func DeleteStatusEventProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &inventory2.StatusEvent[inventory2.StatusEventDeletedBody]{
		AccountId: accountId,
		Type:      inventory2.StatusEventTypeDeleted,
		Body:      inventory2.StatusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
