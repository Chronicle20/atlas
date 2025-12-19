package character

import (
	consumer2 "atlas-npc/kafka/consumer"
	character2 "atlas-npc/kafka/message/character"
	"atlas-npc/shops"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged(db))))
		}
	}
}

func handleStatusEventLogout(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type == character2.StatusEventTypeLogout {
			l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
			_ = shops.NewProcessor(l, ctx, db).ExitAndEmit(e.CharacterId)
		}
	}
}

func handleStatusEventMapChanged(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
		if e.Type == character2.StatusEventTypeMapChanged {
			l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
			_ = shops.NewProcessor(l, ctx, db).ExitAndEmit(e.CharacterId)
		}
	}
}

func handleStatusEventChannelChanged(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
		if e.Type == character2.StatusEventTypeChannelChanged {
			l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
			_ = shops.NewProcessor(l, ctx, db).ExitAndEmit(e.CharacterId)
		}
	}
}
