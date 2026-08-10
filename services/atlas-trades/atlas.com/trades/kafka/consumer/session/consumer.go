// Package session consumes EVENT_TOPIC_SESSION_STATUS and ends the trade of a
// character whose session was destroyed — a socket that dropped without a clean
// logout, or a server-side kick.
//
// It is the belt to the character-status LOGOUT consumer's braces: a client that
// vanishes mid-trade may never produce a LOGOUT, and a trade room left standing
// keeps both sides' inventory reservations refreshed forever (design §5.3 — the
// TTL is a backstop only while the process believes the room is dead).
package session

import (
	consumer2 "atlas-trades/kafka/consumer"
	sessionKafka "atlas-trades/kafka/message/session"
	"atlas-trades/trade"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_session_status")(sessionKafka.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
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

// handleStatusEventDestroyed ends the trade of a character whose session died,
// telling the survivor TRADE_CANCELLED (design §3.3's disconnect row).
//
// It discriminates on the envelope's `type` before the body is used, and then
// skips a session with no character: the login server destroys sessions that
// never selected a character, and character id 0 is not a trader — it is the
// zero value, which would otherwise be looked up in the member index.
func handleStatusEventDestroyed(db *gorm.DB) message.Handler[sessionKafka.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e sessionKafka.StatusEvent) {
		if e.Type != sessionKafka.EventSessionStatusTypeDestroyed {
			return
		}
		if e.CharacterId == 0 {
			return
		}
		l.Debugf("SESSION_DESTROYED for character [%d] account [%d] worldId [%d] channelId [%d]. Tearing down their trade room.", e.CharacterId, e.AccountId, e.WorldId, e.ChannelId)
		if err := trade.NewProcessor(l, ctx, db).TeardownCharacter(uuid.New(), character.Id(e.CharacterId), trade.ReasonTradeCancelled); err != nil {
			l.WithError(err).Errorf("Unable to tear down the trade room of character [%d] on session destroy.", e.CharacterId)
		}
	}
}
