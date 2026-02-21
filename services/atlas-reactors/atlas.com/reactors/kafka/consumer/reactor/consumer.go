package reactor

import (
	consumer2 "atlas-reactors/kafka/consumer"
	"atlas-reactors/reactor"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("reactor_command")(reactor.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(reactor.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreate))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleHit))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyInField))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreate(l logrus.FieldLogger, ctx context.Context, c reactor.Command[reactor.CreateCommandBody]) {
	if c.Type != reactor.CommandTypeCreate {
		return
	}

	t := tenant.MustFromContext(ctx)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	b := reactor.NewModelBuilder(t, f, c.Body.Classification, c.Body.Name).
		SetState(c.Body.State).
		SetPosition(c.Body.X, c.Body.Y).
		SetDelay(c.Body.Delay).
		SetDirection(c.Body.Direction)

	err := reactor.Create(l)(ctx)(b)
	if err != nil {
		l.WithError(err).Errorf("Failed to create reactor for classification [%d].", c.Body.Classification)
	}
}

func handleHit(l logrus.FieldLogger, ctx context.Context, c reactor.Command[reactor.HitCommandBody]) {
	if c.Type != reactor.CommandTypeHit {
		return
	}

	err := reactor.Hit(l)(ctx)(c.Body.ReactorId, c.Body.CharacterId, c.Body.SkillId)
	if err != nil {
		l.WithError(err).Errorf("Failed to process hit for reactor [%d].", c.Body.ReactorId)
	}
}

func handleDestroyInField(l logrus.FieldLogger, ctx context.Context, c reactor.Command[reactor.DestroyInFieldCommandBody]) {
	if c.Type != reactor.CommandTypeDestroyInField {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	reactor.DestroyInField(l)(ctx)(f)
}
