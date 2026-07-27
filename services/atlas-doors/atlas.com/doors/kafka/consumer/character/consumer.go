package character

import (
	enginedoor "atlas-doors/door"
	consumer2 "atlas-doors/kafka/consumer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
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
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
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

func handleLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventLogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	_ = enginedoor.NewProcessor(l, ctx).RemoveByOwner(e.CharacterId, enginedoor.RemoveReasonLogout)
}

func handleChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChangeChannelEventLoginBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	_ = enginedoor.NewProcessor(l, ctx).RemoveByOwner(e.CharacterId, enginedoor.RemoveReasonChannelChanged)
}

func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusEventMapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
	_ = enginedoor.NewProcessor(l, ctx).RemoveByOwnerIfLeftField(e.CharacterId, f)
}
