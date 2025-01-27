package character

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("character_status")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
		}
	}
}

func handleStatusEventLogin(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLoginBody]) {
		if event.Type != EventCharacterStatusTypeLogin {
			return
		}

		err := guild.UpdateMemberOnline(l)(ctx)(db)(event.CharacterId, true)
		if err != nil {
			l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
		}
	}
}

func handleStatusEventLogout(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventLogoutBody]) {
		if event.Type != EventCharacterStatusTypeLogout {
			return

		}

		err := guild.UpdateMemberOnline(l)(ctx)(db)(event.CharacterId, false)
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
		}
	}
}
