package session

import (
	"atlas-mini-games/game"
	consumer2 "atlas-mini-games/kafka/consumer"
	sessionKafka "atlas-mini-games/kafka/message/session"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("session_status")(sessionKafka.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(sessionKafka.EnvEventTopicSessionStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleStatusEventDestroyed tears down whatever mini-game room the character
// occupies when their session is destroyed (client disconnect / server-side
// kick), on the same forfeit-then-leave path as an explicit LEAVE command.
func handleStatusEventDestroyed(db *gorm.DB) message.Handler[sessionKafka.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e sessionKafka.StatusEvent) {
		if e.Type != sessionKafka.EventSessionStatusTypeDestroyed {
			return
		}
		if e.CharacterId == 0 {
			return
		}
		l.Debugf("SESSION_DESTROYED for character [%d] account [%d] world [%d] channel [%d]. Tearing down mini-game membership.", e.CharacterId, e.AccountId, e.WorldId, e.ChannelId)
		if err := game.NewProcessor(l, ctx, db).TeardownCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to tear down mini-game membership for character [%d] on session destroy.", e.CharacterId)
		}
	}
}
