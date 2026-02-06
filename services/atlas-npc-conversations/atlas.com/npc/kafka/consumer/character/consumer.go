package character

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	"atlas-npc-conversations/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(db))))
	}
}

func handleStatusEventLogout(db *gorm.DB) message.Handler[character.StatusEvent[character.StatusEventLogoutBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.StatusEventLogoutBody]) {
		if e.Type != character.StatusEventTypeLogout {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).End(e.CharacterId)
	}
}

func handleStatusEventChannelChanged(db *gorm.DB) message.Handler[character.StatusEvent[character.StatusEventChannelChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.StatusEventChannelChangedBody]) {
		if e.Type != character.StatusEventTypeChannelChanged {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).End(e.CharacterId)
	}
}

func handleStatusEventMapChanged(db *gorm.DB) message.Handler[character.StatusEvent[character.StatusEventMapChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.StatusEventMapChangedBody]) {
		if e.Type != character.StatusEventTypeMapChanged {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).End(e.CharacterId)
	}
}
