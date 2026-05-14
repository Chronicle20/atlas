package data

import (
	consumer2 "atlas-maps/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(
				consumer2.NewConfig(l)("data_events")(EnvEventTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.LastOffset),
			)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		if !consumerEnabled() {
			l.Infof("DATA_EVENTS_CONSUMER_ENABLED=false; not registering DATA_UPDATED handler.")
			return nil
		}
		t, _ := topic.EnvProvider(l)(EnvEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDataUpdated))); err != nil {
			return err
		}
		return nil
	}
}
