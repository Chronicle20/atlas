package producer

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

type Provider func(token string) producer.MessageProducer

func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
	return func(ctx context.Context) func(token string) producer.MessageProducer {
		sd := producer.SpanHeaderDecorator(ctx)
		td := producer.TenantHeaderDecorator(ctx)
		return func(token string) producer.MessageProducer {
			return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
