package character

import (
	consumer2 "atlas-skills/kafka/consumer"
	"atlas-skills/kafka/message/character"
	"atlas-skills/macro"
	"atlas-skills/skill"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db))))
		}
	}
}

func handleStatusEventLogout(db *gorm.DB) message.Handler[character.StatusEvent[character.LogoutStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.LogoutStatusEventBody]) {
		if e.Type != character.EventStatusTypeLogout {
			return
		}
		err := skill.NewProcessor(l, ctx, db).ClearAll(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", e.CharacterId)
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[character.StatusEvent[character.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.DeletedStatusEventBody]) {
		if e.Type != character.StatusEventTypeDeleted {
			return
		}

		err := skill.NewProcessor(l, ctx, db).ClearAll(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete for character [%d].", e.CharacterId)
		}
		err = skill.NewProcessor(l, ctx, db).Delete(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete for character [%d].", e.CharacterId)
		}
		err = macro.NewProcessor(l, ctx, db).Delete(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete for character [%d].", e.CharacterId)
		}
	}
}
