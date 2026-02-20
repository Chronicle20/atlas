package character

import (
	"atlas-chairs/chair"
	"atlas-chairs/character"
	consumer2 "atlas-chairs/kafka/consumer"
	character2 "atlas-chairs/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("character_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
	if e.Type == character2.EventCharacterStatusTypeLogin {
		l.Debugf("Character [%d] has logged in. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).Build()
		character.NewProcessor(l, ctx).Enter(f, e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type == character2.EventCharacterStatusTypeLogout {
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).Build()
		character.NewProcessor(l, ctx).Exit(f, e.CharacterId)
		_ = chair.NewProcessor(l, ctx).Clear(f, e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type == character2.EventCharacterStatusTypeMapChanged {
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
		of := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.OldMapId).Build()
		nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).Build()
		character.NewProcessor(l, ctx).TransitionMap(of, nf, e.CharacterId)
		_ = chair.NewProcessor(l, ctx).Clear(of, e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if e.Type == character2.EventCharacterStatusTypeChannelChanged {
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d].", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
		of := field.NewBuilder(e.WorldId, e.Body.OldChannelId, e.Body.MapId).Build()
		nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).Build()
		character.NewProcessor(l, ctx).TransitionChannel(of, nf, e.CharacterId)
		_ = chair.NewProcessor(l, ctx).Clear(nf, e.CharacterId)
	}
}
