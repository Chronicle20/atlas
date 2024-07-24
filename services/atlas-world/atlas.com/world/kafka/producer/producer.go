package producer

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type Provider func(token string) producer.MessageProducer

func ProviderImpl(l logrus.FieldLogger) func(span opentracing.Span) func(token string) producer.MessageProducer {
	return func(span opentracing.Span) func(token string) producer.MessageProducer {
		return func(token string) producer.MessageProducer {
			return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(token)))(producer.SpanHeaderDecorator(span))
		}
	}
}
