package chalkboard

import (
	consumer2 "atlas-channel/kafka/consumer"
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	charpkt "github.com/Chronicle20/atlas-packet/character/clientbound"
	statpkt "github.com/Chronicle20/atlas-packet/stat/clientbound"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chalkboard_status_event")(chalkboard2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(chalkboard2.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetCommand(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClearCommand(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleSetCommand(sc server.Model, wp writer.Producer) message.Handler[chalkboard2.StatusEvent[chalkboard2.SetStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chalkboard2.StatusEvent[chalkboard2.SetStatusEventBody]) {
		if e.Type != chalkboard2.EventTopicStatusTypeSet {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(charpkt.ChalkboardUseWriter)(charpkt.NewChalkboardUse(e.CharacterId, e.Body.Message).Encode)(s)
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(charpkt.ChalkboardUseWriter)(charpkt.NewChalkboardClear(e.CharacterId).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to show chalkboard clear by character [%d].", e.CharacterId)
		}
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, enableActions(l)(ctx)(wp))
		if err != nil {
			l.WithError(err).Errorf("Unable to enable actions for character [%d].", e.CharacterId)
		}
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)
		}
	}
}
