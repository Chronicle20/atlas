package character

import (
	consumer2 "atlas-transports/kafka/consumer"
	"atlas-transports/instance"
	character2 "atlas-transports/kafka/message/character"
	"atlas-transports/transport"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	message2 "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicStatus)()
		if _, err := rf(t, message2.AdaptHandler(message2.PersistentConfig(handleLogoutEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message2.AdaptHandler(message2.PersistentConfig(handleLoginEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleLogoutEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LogoutStatusEventBody]) {
	if e.Type != character2.StatusEventTypeLogout {
		return
	}

	l.Debugf("Character [%d] logged out in map [%d].", e.CharacterId, e.Body.MapId)

	// Handle instance transport logout (remove from active transport)
	_ = instance.NewProcessor(l, ctx).HandleLogoutAndEmit(e.CharacterId, e.WorldId, e.Body.ChannelId)

	// Warp character to route start map if they logged out in a scheduled transport map
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	_ = transport.NewProcessor(l, ctx).WarpToRouteStartMapOnLogoutAndEmit(e.CharacterId, f)
}

func handleLoginEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LoginStatusEventBody]) {
	if e.Type != character2.StatusEventTypeLogin {
		return
	}

	l.Debugf("Character [%d] logged in at map [%d].", e.CharacterId, e.Body.MapId)

	// Crash recovery: check if character logged in at a transit map and warp to start
	_ = instance.NewProcessor(l, ctx).HandleLoginAndEmit(e.CharacterId, e.Body.MapId, e.WorldId, e.Body.ChannelId)
}
