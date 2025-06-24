package character

import (
	consumer2 "atlas-transports/kafka/consumer"
	character2 "atlas-transports/kafka/message/character"
	"atlas-transports/transport"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicStatus)()
		_, _ = rf(t, message2.AdaptHandler(message2.PersistentConfig(handleEventStatus)))
	}
}

func handleEventStatus(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LogoutStatusEventBody]) {
	if e.Type != character2.StatusEventTypeLogout {
		return
	}

	l.Debugf("Character [%d] logged out in map [%d].", e.CharacterId, e.Body.MapId)

	// Warp character to route start map if they logged out in a transport map
	f := field.NewBuilder(world.Id(e.WorldId), channel.Id(e.Body.ChannelId), map2.Id(e.Body.MapId)).Build()
	_ = transport.NewProcessor(l, ctx).WarpToRouteStartMapOnLogoutAndEmit(e.CharacterId, f)
}
