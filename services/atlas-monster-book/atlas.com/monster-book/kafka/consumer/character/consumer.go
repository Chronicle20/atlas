package character

import (
	"context"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	consumer2 "atlas-monster-book/kafka/consumer"
	characterMsg "atlas-monster-book/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status")(characterMsg.EnvEventTopicStatus)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]) {
		if e.Type != characterMsg.StatusEventTypeDeleted {
			return
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			cp := card.NewProcessor(l, ctx, tx)
			colp := collection.NewProcessor(l, ctx, tx)
			if err := cp.DeleteByCharacterId(e.CharacterId); err != nil {
				return err
			}
			return colp.DeleteByCharacterId(e.CharacterId)
		}); err != nil {
			l.WithError(err).Errorf("Cascading monster-book delete failed for character %d.", e.CharacterId)
		}
	}
}
