package character

import (
	consumer2 "atlas-consumables/kafka/consumer"
	character2 "atlas-consumables/kafka/message/character"
	"atlas-consumables/map/character"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
	if e.Type == character2.EventCharacterStatusTypeLogin {
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
		l.Debugf("Character [%d] has logged into field [%s].", e.CharacterId, f.Id())
		character.NewProcessor(l, ctx).Enter(f, e.CharacterId)
		return
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type == character2.EventCharacterStatusTypeLogout {
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
		l.Debugf("Character [%d] has logged out of field [%s].", e.CharacterId, f.Id())
		character.NewProcessor(l, ctx).Exit(f, e.CharacterId)
		return
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type == character2.EventCharacterStatusTypeMapChanged {
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
		l.Debugf("Character [%d] has changed maps. field [%s]. oldMapId [%d].", e.CharacterId, f.Id(), e.Body.OldMapId)
		character.NewProcessor(l, ctx).TransitionMap(f, e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if e.Type == character2.EventCharacterStatusTypeChannelChanged {
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
		l.Debugf("Character [%d] has changed channels. field [%s]. oldChannelId [%d].", e.CharacterId, f.Id(), e.Body.OldChannelId)
		character.NewProcessor(l, ctx).TransitionChannel(f, e.CharacterId)
	}
}
