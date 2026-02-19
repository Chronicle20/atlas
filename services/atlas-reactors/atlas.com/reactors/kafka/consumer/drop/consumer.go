package drop

import (
	consumer2 "atlas-reactors/kafka/consumer"
	dropMessage "atlas-reactors/kafka/message/drop"
	"atlas-reactors/reactor"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_status_event")(dropMessage.EnvEventTopicDropStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(dropMessage.EnvEventTopicDropStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated)))
	}
}

func handleStatusEventCreated(l logrus.FieldLogger, ctx context.Context, e dropMessage.StatusEvent[dropMessage.StatusEventCreatedBody]) {
	if e.Type != dropMessage.StatusEventTypeCreated {
		return
	}

	if !e.Body.PlayerDrop {
		return
	}

	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	reactor.ActivateItemReactors(l)(ctx)(e.DropId, e.Body.ItemId, e.Body.Quantity, e.Body.X, e.Body.Y, e.Body.OwnerId, f)
}
