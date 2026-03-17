package merchant

import (
	_map "atlas-channel/map"
	consumer2 "atlas-channel/kafka/consumer"
	merchant2 "atlas-channel/kafka/message/merchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	mapId "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	interactionpkt "github.com/Chronicle20/atlas-packet/interaction"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("merchant_status_event")(merchant2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(merchant2.EnvStatusEventTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShopOpenedEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShopClosedEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMessageSentEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleShopOpenedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventShopOpenedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventShopOpenedBody]) {
		if e.Type != merchant2.StatusEventShopOpened {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.Body.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Merchant shop [%s] opened by character [%d] in map [%d] instance [%s].", e.Body.ShopId, e.CharacterId, e.Body.MapId, e.Body.InstanceId)

		f := field.NewBuilder(e.Body.WorldId, e.Body.ChannelId, mapId.Id(e.Body.MapId)).SetInstance(e.Body.InstanceId).Build()
		mr := &interactionpkt.MiniRoomBase{
			MiniRoomTypeVal: interactionpkt.MerchantShopMiniRoomType,
			Title:           e.Body.Title,
			CapacityVal:     4,
			OwnerId:         e.CharacterId,
			VisitorList:     []interactionpkt.MiniRoomVisitor{},
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(e.CharacterId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn merchant shop [%s] for characters in map [%d] instance [%s].", e.Body.ShopId, e.Body.MapId, e.Body.InstanceId)
		}
	}
}

func handleShopClosedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventShopClosedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventShopClosedBody]) {
		if e.Type != merchant2.StatusEventShopClosed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Merchant shop [%s] closed by character [%d].", e.Body.ShopId, e.CharacterId)

		mr := &interactionpkt.MiniRoomBase{}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Despawn(e.CharacterId)))
	}
}

func handleMessageSentEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventMessageSentBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventMessageSentBody]) {
		if e.Type != merchant2.StatusEventMessageSent {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Unable to parse shop id [%s].", e.Body.ShopId)
			return
		}

		l.Debugf("Message sent in shop [%s] by character [%d] slot [%d].", shopId, e.Body.CharacterId, e.Body.Slot)

		// TODO broadcast InteractionChat to all visitors in the shop session
		// This requires knowing which sessions are in the shop, which will be
		// tracked once the full mini-room session management is implemented.
		_ = shopId
	}
}
