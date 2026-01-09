package saga

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// CreateCommandProvider creates a Kafka message provider for initiating a saga
func CreateCommandProvider(s Saga) model.Provider[[]kafka.Message] {
	// Use transaction ID as the key to ensure ordering
	key := producer.CreateKey(int(s.TransactionId.ID()))
	value := &s
	return producer.SingleMessageProvider(key, value)
}
