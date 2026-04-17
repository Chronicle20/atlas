package saga

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(s sharedsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
