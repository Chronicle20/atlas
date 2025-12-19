package reactor

import (
	consumer2 "atlas-reactors/kafka/consumer"
	"atlas-reactors/reactor"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("reactor_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreate)))
	}
}

func handleCreate(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
	if c.Type != CommandTypeCreate {
		return
	}

	t := tenant.MustFromContext(ctx)
	b := reactor.NewModelBuilder(t, c.WorldId, c.ChannelId, c.MapId, c.Body.Classification, c.Body.Name).
		SetState(c.Body.State).
		SetPosition(c.Body.X, c.Body.Y).
		SetDelay(c.Body.Delay).
		SetDirection(c.Body.Direction)

	_ = reactor.Create(l)(ctx)(b)
}
