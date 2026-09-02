package dragon

import (
	consumer2 "atlas-dragons/kafka/consumer"
	"context"

	dragonstate "atlas-dragons/dragon"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("dragon_command")(EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, err := topic.EnvProvider(l)(EnvCommandTopic)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c Command[CreateCommandBody]) {
	if c.Type != CommandTypeCreate {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	if err := dragonstate.NewProcessor(l, ctx).Create(f, c.Body.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon for character [%d].", c.Body.CharacterId)
	}
}

func handleDestroyCommand(l logrus.FieldLogger, ctx context.Context, c Command[DestroyCommandBody]) {
	if c.Type != CommandTypeDestroy {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(c.Body.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon for character [%d].", c.Body.CharacterId)
	}
}

func handleMoveCommand(l logrus.FieldLogger, ctx context.Context, c Command[MoveCommandBody]) {
	if c.Type != CommandTypeMove {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Move(c.Body.CharacterId, c.Body.StartX, c.Body.StartY, c.Body.Stance, c.Body.RawMovement); err != nil {
		l.WithError(err).Errorf("Failed to move dragon for character [%d].", c.Body.CharacterId)
	}
}
