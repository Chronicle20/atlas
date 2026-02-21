package saga

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas-saga"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
