package character

import (
	"atlas-pets/character"
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
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
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterDeleted(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged)))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
		}
	}
}

func handleCharacterDeleted(db *gorm.DB) message.Handler[statusEvent[statusEventDeletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDeletedBody]) {
		if e.Type != StatusEventTypeDeleted {
			return
		}

		_ = pet.DeleteForCharacter(l)(ctx)(db)(e.CharacterId)
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventLoginBody]) {
	if e.Type == StatusEventTypeLogin {
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		character.Enter(ctx)(e.WorldId, e.Body.ChannelId, e.Body.MapId, e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventLogoutBody]) {
	if e.Type == StatusEventTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		character.Exit(ctx)(e.WorldId, e.Body.ChannelId, e.Body.MapId, e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMapChangedBody]) {
	if e.Type == StatusEventTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
		character.TransitionMap(ctx)(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId, e.CharacterId, e.Body.OldMapId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e statusEvent[changeChannelEventLoginBody]) {
	if e.Type == StatusEventTypeChannelChanged {
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
		character.TransitionChannel(ctx)(e.WorldId, e.Body.ChannelId, e.Body.OldChannelId, e.CharacterId, e.Body.MapId)
	}
}
