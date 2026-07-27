package outbox

import (
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// EmitProvider returns a producer.Provider-shaped value whose
// MessageProducer persists messages as outbox rows inside tx instead of
// writing to Kafka. The return type is the unnamed func type underlying
// every service-local kafka/producer.Provider, so existing message.Emit /
// EmitWithResult call sites accept it without conversion. Topic tokens are
// env-resolved and span+tenant headers applied from ctx at enqueue time;
// the drainer publishes after the transaction commits.
func EmitProvider(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) func(token string) kafkaproducer.MessageProducer {
	return func(token string) kafkaproducer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				return err
			}
			return EnqueueBuffer(l, ctx, tx, map[string][]kafka.Message{token: msgs})
		}
	}
}
