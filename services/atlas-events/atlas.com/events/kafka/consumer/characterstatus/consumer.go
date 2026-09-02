// Package characterstatus wires the atlas-character EVENT_TOPIC_CHARACTER_STATUS
// LOGIN event into the Anniversary login-time buff grant. This is a REACTION,
// not a query in the login path: atlas-events being unavailable delays the
// buff, it never delays or fails the login (FR-A8). Consumer plumbing only —
// decode, delegate; the grant logic lives in events/anniversary (login.go).
package characterstatus

import (
	"atlas-events/events/anniversary"
	consumer2 "atlas-events/kafka/consumer"
	characterstatus2 "atlas-events/kafka/message/characterstatus"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(characterstatus2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(characterstatus2.EnvEventTopicCharacterStatus)()
			if err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventLogin(db *gorm.DB) message.Handler[characterstatus2.StatusEvent[characterstatus2.StatusEventLoginBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventLoginBody]) {
		if e.Type != characterstatus2.StatusEventTypeLogin {
			return
		}
		if err := anniversary.NewLoginProcessor(l, ctx, db).OnLogin(e); err != nil {
			l.WithError(err).Errorf("Unable to grant Anniversary buff to character [%d] at login.", e.CharacterId)
		}
	}
}
