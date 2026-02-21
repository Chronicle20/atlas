package character

import (
	consumer2 "atlas-storage/kafka/consumer"
	character2 "atlas-storage/kafka/message/character"
	"atlas-storage/projection"
	"atlas-storage/storage"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged()))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLogout() func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type != character2.StatusEventTypeLogout {
			return
		}

		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].",
			e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)

		// Clean up projection
		cleanupProjection(l, ctx, e.CharacterId)
	}
}

func handleStatusEventChannelChanged() func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
		if e.Type != character2.StatusEventTypeChannelChanged {
			return
		}

		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].",
			e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)

		// Clean up projection
		cleanupProjection(l, ctx, e.CharacterId)
	}
}

func handleStatusEventMapChanged() func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
		if e.Type != character2.StatusEventTypeMapChanged {
			return
		}

		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].",
			e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)

		// Clean up projection on map change as well
		cleanupProjection(l, ctx, e.CharacterId)
	}
}

func cleanupProjection(l logrus.FieldLogger, ctx context.Context, characterId uint32) {
	// Remove NPC context from cache (legacy)
	cache := storage.GetNpcContextCache()
	cache.Remove(characterId)

	// Delete the projection
	projection.GetManager().Delete(ctx, characterId)

	l.Debugf("Cleaned up projection and NPC context for character [%d]", characterId)
}
