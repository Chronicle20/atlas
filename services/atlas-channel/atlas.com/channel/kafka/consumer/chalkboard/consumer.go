package chalkboard

import (
	consumer2 "atlas-channel/kafka/consumer"
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chalkboard_status_event")(chalkboard2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(chalkboard2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSetCommand(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleClearCommand(sc, wp))))
			}
		}
	}
}

func handleSetCommand(sc server.Model, wp writer.Producer) message.Handler[chalkboard2.StatusEvent[chalkboard2.SetStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chalkboard2.StatusEvent[chalkboard2.SetStatusEventBody]) {
		if e.Type != chalkboard2.EventTopicStatusTypeSet {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Map(_map2.Id(e.MapId)), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.ChalkboardUse)(writer.ChalkboardUseBody(e.CharacterId, e.Body.Message))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to show chalkboard in use by character [%d].", e.CharacterId)
		}
	}
}

func handleClearCommand(sc server.Model, wp writer.Producer) message.Handler[chalkboard2.StatusEvent[chalkboard2.ClearStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chalkboard2.StatusEvent[chalkboard2.ClearStatusEventBody]) {
		if e.Type != chalkboard2.EventTopicStatusTypeClear {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Map(_map2.Id(e.MapId)), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.ChalkboardUse)(writer.ChalkboardClearBody(e.CharacterId))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to show chalkboard clear by character [%d].", e.CharacterId)
		}
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, enableActions(l)(ctx)(wp))
		if err != nil {
			l.WithError(err).Errorf("Unable to enable actions for character [%d].", e.CharacterId)
		}
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true))
		}
	}
}
