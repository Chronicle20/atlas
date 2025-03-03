package drop

import (
	"atlas-channel/drop"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
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
			rf(consumer2.NewConfig(l)("drop_status_event")(EnvEventTopicDropStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventPickedUp(sc, wp))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdStatusEventBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
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
			Build()

		err := _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), session.Announce(l)(ctx)(wp)(writer.DropSpawn)(writer.DropSpawnBody(l, tenant.MustFromContext(ctx))(d, writer.DropEnterTypeFresh, 0)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn drop [%d] for characters in map [%d].", d.Id(), e.MapId)
		}
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[statusEvent[expiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[expiredStatusEventBody]) {
		if e.Type != StatusEventTypeExpired {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.DropDestroy)(writer.DropDestroyBody(l, tenant.MustFromContext(ctx))(e.DropId, writer.DropDestroyTypeExpire, s.CharacterId(), -1))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy drop [%d] for characters in map [%d].", e.DropId, e.MapId)
		}
	}
}

func handleStatusEventPickedUp(sc server.Model, wp writer.Producer) message.Handler[statusEvent[pickedUpStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[pickedUpStatusEventBody]) {
		if e.Type != StatusEventTypePickedUp {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		l.Debugf("[%d] is picking up drop [%d].", e.Body.CharacterId, e.DropId)

		go func() {
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
				var bp writer.BodyProducer
				if e.Body.Meso > 0 {
					bp = writer.CharacterStatusMessageOperationDropPickUpMesoBody(l)(false, e.Body.Meso, 0)
				} else if e.Body.EquipmentId > 0 {
					bp = writer.CharacterStatusMessageOperationDropPickUpUnStackableItemBody(l)(e.Body.ItemId)
				} else {
					bp = writer.CharacterStatusMessageOperationDropPickUpStackableItemBody(l)(e.Body.ItemId, e.Body.Quantity)
				}

				err := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(bp)(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to write status message to character [%d] picking up drop [%d].", s.CharacterId(), e.DropId)
				}
				return err
			})
		}()

		go func() {
			err := _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), session.Announce(l)(ctx)(wp)(writer.DropDestroy)(writer.DropDestroyBody(l, tenant.MustFromContext(ctx))(e.DropId, writer.DropDestroyTypePickUp, e.Body.CharacterId, -1)))
			if err != nil {
				l.WithError(err).Errorf("Unable to pick up drop [%d] for characters in map [%d].", e.DropId, e.MapId)
			}
		}()
	}
}
