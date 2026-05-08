package producer

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/sirupsen/logrus"
)

// Provider returns a MessageProducer for a given topic env-var token.
type Provider func(token string) producer.MessageProducer

// ProviderImpl mirrors atlas-data: each call yields a producer with the
// span+tenant header decorators attached.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
	return func(ctx context.Context) func(token string) producer.MessageProducer {
		sd := producer.SpanHeaderDecorator(ctx)
		td := producer.TenantHeaderDecorator(ctx)
		return func(token string) producer.MessageProducer {
			return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
