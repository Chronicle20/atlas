package drop

import (
	"atlas-channel/drop"
	consumer2 "atlas-channel/kafka/consumer"
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
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_status_event")(EnvEventTopicDropStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicDropStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdStatusEventBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		d := drop.NewModelBuilder().
			SetId(e.DropId).
			SetItem(e.Body.ItemId, e.Body.Quantity).
			SetMeso(e.Body.Meso).
			SetType(e.Body.Type).
			SetPosition(e.Body.X, e.Body.Y).
			SetOwner(e.Body.OwnerId, e.Body.OwnerPartyId).
			SetDropper(e.Body.DropperUniqueId, e.Body.DropperX, e.Body.DropperY).
			SetPlayerDrop(e.Body.PlayerDrop).
			SetMod(e.Body.Mod).
			Build()

		_ = _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), e.MapId, func(s session.Model) error {
			l.Debugf("Spawning [%d] drop [%d] for character [%d].", d.ItemId(), d.Id(), s.CharacterId())
			err := session.Announce(l)(ctx)(wp)(writer.DropSpawn)(s, writer.DropSpawnBody(l, tenant.MustFromContext(ctx))(d, writer.DropEnterTypeFresh, 0))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn drop [%d] for character [%d].", d.Id(), s.CharacterId())
			}
			return err
		})
	}
}
