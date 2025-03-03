package messenger

import (
	consumer2 "atlas-messengers/kafka/consumer"
	"atlas-messengers/messenger"
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
			rf(consumer2.NewConfig(l)("messenger_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreate)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeave)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestInvite)))
	}
}

func handleCreate(l logrus.FieldLogger, ctx context.Context, c commandEvent[createCommandBody]) {
	if c.Type != CommandMessengerCreate {
		return
	}
	_, err := messenger.Create(l)(ctx)(c.ActorId)
	if err != nil {
		l.WithError(err).Errorf("Unable to create messenger for leader [%d].", c.ActorId)
	}
}

func handleJoin(l logrus.FieldLogger, ctx context.Context, c commandEvent[joinCommandBody]) {
	if c.Type != CommandMessengerJoin {
		return
	}
	_, err := messenger.Join(l)(ctx)(c.Body.MessengerId, c.ActorId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to join messenger [%d].", c.ActorId, c.Body.MessengerId)
	}
}

func handleLeave(l logrus.FieldLogger, ctx context.Context, c commandEvent[leaveCommandBody]) {
	if c.Type != CommandMessengerLeave {
		return
	}

	_, err := messenger.Leave(l)(ctx)(c.Body.MessengerId, c.ActorId)
	if err != nil {
		l.WithError(err).Errorf("Unable to leave messenger [%d].", c.Body.MessengerId)
		return
	}
}

func handleRequestInvite(l logrus.FieldLogger, ctx context.Context, c commandEvent[requestInviteBody]) {
	if c.Type != CommandMessengerRequestInvite {
		return
	}
	err := messenger.RequestInvite(l)(ctx)(c.ActorId, c.Body.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to invite [%d] to messenger.", c.Body.CharacterId)
	}
}
