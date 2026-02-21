package _map

import (
	"atlas-expressions/expression"
	consumer2 "atlas-expressions/kafka/consumer"
	mapMsg "atlas-expressions/kafka/message/map"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(mapMsg.EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(mapMsg.EnvEventTopicMapStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, ctx context.Context, event mapMsg.StatusEvent[mapMsg.CharacterExit]) {
	if event.Type != mapMsg.EventTopicMapStatusTypeCharacterExit {
		return
	}
	processor := expression.NewProcessor(l, ctx)
	_, _ = processor.ClearAndEmit(event.TransactionId, event.Body.CharacterId)
}
