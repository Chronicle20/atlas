package saga

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(s Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
