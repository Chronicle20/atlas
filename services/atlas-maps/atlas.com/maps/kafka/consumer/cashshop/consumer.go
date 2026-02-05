package cashshop

import (
	consumer2 "atlas-maps/kafka/consumer"
	cashshopKafka "atlas-maps/kafka/message/cashshop"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(cashshopKafka.EnvEventTopicCashShopStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(cashshopKafka.EnvEventTopicCashShopStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventEnter)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExit)))
	}
}

func handleStatusEventEnter(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	if event.Type == cashshopKafka.EventCashShopStatusTypeCharacterEnter {
		l.Debugf("Character [%d] has entered cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
		_ = p.ExitAndEmit(transactionId, f, event.Body.CharacterId)
		return
	}
}

func handleStatusEventExit(l logrus.FieldLogger, ctx context.Context, event cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]) {
	if event.Type == cashshopKafka.EventCashShopStatusTypeCharacterExit {
		l.Debugf("Character [%d] has exited cash shop.", event.Body.CharacterId)
		transactionId := uuid.New()
		f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).Build()
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
		_ = p.EnterAndEmit(transactionId, f, event.Body.CharacterId)
		return
	}
}
