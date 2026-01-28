package saga

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	scriptsaga "github.com/Chronicle20/atlas-script-core/saga"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(s scriptsaga.Saga) model.Provider[[]kafka.Message] {
	key := []byte(s.TransactionId.String())
	return producer.SingleMessageProvider(key, &s)
}
