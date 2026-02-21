package character

import (
	"atlas-family/family"
	consumer2 "atlas-family/kafka/consumer"
	character2 "atlas-family/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventDeletedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventDeletedBody]) {
		if e.Type != character2.EventCharacterStatusTypeDeleted {
			return
		}

		_, err := family.NewProcessor(l, ctx, db).RemoveMemberAndEmit(uuid.New(), e.CharacterId, "CHARACTER_DELETED")()
		if err != nil {
			// ErrMemberNotFound is expected if character was not in a family
			l.WithError(err).Debugf("Unable to remove deleted character [%d] from family (may not be a member).", e.CharacterId)
		}
	}
}
