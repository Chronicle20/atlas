package saga

import (
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateCommandProvider keys the saga command by its transaction id so all
// commands for a saga land on the same partition (ordered processing).
func CreateCommandProvider(s Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
