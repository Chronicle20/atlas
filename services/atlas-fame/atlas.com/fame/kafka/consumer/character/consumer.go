package character

import (
	"atlas-fame/fame"
	consumer2 "atlas-fame/kafka/consumer"
	character2 "atlas-fame/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
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

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db))))
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventDeletedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventDeletedBody]) {
		if e.Type != character2.EventCharacterStatusTypeDeleted {
			return
		}

		err := fame.NewProcessor(l, ctx, db).DeleteByCharacterId(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete fame records for character [%d].", e.CharacterId)
		}
	}
}
