package character

import (
	consumer2 "atlas-summons/kafka/consumer"
	"atlas-summons/summon"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChannelChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChanged))); err != nil {
			return err
		}
		return nil
	}
}

// handleLogout / handleChannelChanged / handleMapChanged all despawn every summon
// owned by the character. Summons are bound to the field at spawn and must not
// follow across logout, channel change, or map change; re-cast in the new field
// is the player's responsibility (Cosmic Character.java:3769-3791).
func handleLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	_ = summon.NewProcessor(l, ctx).DespawnAllForOwner(e.CharacterId)
}

func handleChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChannelChangedBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	_ = summon.NewProcessor(l, ctx).DespawnAllForOwner(e.CharacterId)
}

func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	_ = summon.NewProcessor(l, ctx).DespawnAllForOwner(e.CharacterId)
}
