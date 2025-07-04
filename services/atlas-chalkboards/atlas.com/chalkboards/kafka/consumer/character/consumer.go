package character

import (
	"atlas-chalkboards/chalkboard"
	"atlas-chalkboards/character"
	consumer2 "atlas-chalkboards/kafka/consumer"
	character2 "atlas-chalkboards/kafka/message/character"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		f := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), _map.Id(e.Body.MapId)).Build()
		character.NewProcessor(l, ctx).Enter(f, e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type == character2.EventCharacterStatusTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		f := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), _map.Id(e.Body.MapId)).Build()
		character.NewProcessor(l, ctx).Exit(f, e.CharacterId)
		_ = chalkboard.NewProcessor(l, ctx).Clear(f, e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type == character2.EventCharacterStatusTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
		of := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), _map.Id(e.Body.OldMapId)).Build()
		nf := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), _map.Id(e.Body.TargetMapId)).Build()
		character.NewProcessor(l, ctx).TransitionMap(of, nf, e.CharacterId)
		_ = chalkboard.NewProcessor(l, ctx).Clear(of, e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if e.Type == character2.EventCharacterStatusTypeChannelChanged {
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
		of := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.OldChannelId), _map.Id(e.Body.MapId)).Build()
		nf := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), _map.Id(e.Body.MapId)).Build()
		character.NewProcessor(l, ctx).TransitionChannel(of, nf, e.CharacterId)
		_ = chalkboard.NewProcessor(l, ctx).Clear(of, e.CharacterId)
	}
}
