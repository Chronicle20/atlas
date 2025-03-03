package cashshop

import (
	consumer2 "atlas-maps/kafka/consumer"
	_map "atlas-maps/map"
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
			rf(consumer2.NewConfig(l)("status_event")(EnvEventTopicCashShopStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCashShopStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventEnter)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExit)))
	}
}

func handleStatusEventEnter(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterMovementBody]) {
	if event.Type == EventCashShopStatusTypeCharacterEnter {
		l.Debugf("Character [%d] has entered cash shop.", event.Body.CharacterId)
		_map.Exit(l)(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.Body.CharacterId)
		return
	}
}

func handleStatusEventExit(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterMovementBody]) {
	if event.Type == EventCashShopStatusTypeCharacterExit {
		l.Debugf("Character [%d] has exited cash shop.", event.Body.CharacterId)
		_map.Enter(l)(ctx)(event.WorldId, event.Body.ChannelId, event.Body.MapId, event.Body.CharacterId)
		return
	}
}
