package session

import (
	"context"

	consumer2 "atlas-maps/kafka/consumer"
	sessionKafka "atlas-maps/kafka/message/session"
	"atlas-maps/kafka/producer"
	timer "atlas-maps/map/timer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// ForceReturner is the seam used to test the handler without standing up a
// real timer Processor. Production binds it to timer.Processor's
// ForceReturnIfTracked.
type ForceReturner interface {
	ForceReturnIfTracked(characterId uint32) bool
}

type forceReturnerProvider func(ctx context.Context) ForceReturner

func defaultForceReturnerProvider(l logrus.FieldLogger) forceReturnerProvider {
	return func(ctx context.Context) ForceReturner {
		return timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
	}
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("session_status")(sessionKafka.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(sessionKafka.EnvEventTopicSessionStatus)()
		if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(newHandleSessionDestroyed(defaultForceReturnerProvider(l))))); err != nil {
			return err
		}
		return nil
	}
}

func newHandleSessionDestroyed(provider forceReturnerProvider) kafkaMessage.Handler[sessionKafka.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e sessionKafka.StatusEvent) {
		if e.Type != sessionKafka.EventSessionStatusTypeDestroyed {
			return
		}
		if e.CharacterId == 0 {
			return
		}
		l.Debugf("SESSION_DESTROYED for character [%d] account [%d] world [%d] channel [%d].", e.CharacterId, e.AccountId, e.WorldId, e.ChannelId)
		provider(ctx).ForceReturnIfTracked(e.CharacterId)
	}
}
