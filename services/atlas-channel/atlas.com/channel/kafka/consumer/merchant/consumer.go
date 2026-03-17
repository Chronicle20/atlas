package merchant

import (
	_map "atlas-channel/map"
	"atlas-channel/merchant"
	consumer2 "atlas-channel/kafka/consumer"
	merchant2 "atlas-channel/kafka/message/merchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	mapId "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	interactionpkt "github.com/Chronicle20/atlas-packet/interaction"
	merchantpkt "github.com/Chronicle20/atlas-packet/merchant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("merchant_status_event")(merchant2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("merchant_listing_event")(merchant2.EnvListingEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleVisitorEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCapacityFullEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePurchaseFailedEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleFrederickNotificationEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMessageSentEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMaintenanceEvent(sc, wp)))); err != nil {
					return err
				}

				t, _ = topic.EnvProvider(l)(merchant2.EnvListingEventTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleListingPurchasedEvent(sc, wp)))); err != nil {
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

func handleVisitorEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		sp := session.NewProcessor(l, ctx)

		switch e.Type {
		case merchant2.StatusEventVisitorEntered:
			l.Debugf("Visitor [%d] entered shop [%s].", e.Body.CharacterId, e.Body.ShopId)
			broadcastToShopViewers(l, ctx, sc, wp, e.Body.ShopId, func(characterIds []uint32) {
				announce := session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)
				for _, cid := range characterIds {
					if cid == e.Body.CharacterId {
						continue
					}
					_ = sp.IfPresentByCharacterId(sc.Channel())(cid, announce(interactionpkt.CharacterInteractionChatBody(0, fmt.Sprintf("Visitor [%d] has entered.", e.Body.CharacterId))))
				}
			})
		case merchant2.StatusEventVisitorExited:
			l.Debugf("Visitor [%d] exited shop [%s].", e.Body.CharacterId, e.Body.ShopId)
		case merchant2.StatusEventVisitorEjected:
			l.Debugf("Visitor [%d] ejected from shop [%s].", e.Body.CharacterId, e.Body.ShopId)
			_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)(interactionpkt.CharacterInteractionEnterResultErrorBody(interactionpkt.CharacterInteractionEnterErrorModeRoomClosed)))
		}
	}
}

func handleMaintenanceEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		switch e.Type {
		case merchant2.StatusEventMaintenanceEntered:
			l.Debugf("Maintenance entered for shop [%s] by character [%d].", e.Body.ShopId, e.Body.CharacterId)
		case merchant2.StatusEventMaintenanceExited:
			l.Debugf("Maintenance exited for shop [%s] by character [%d].", e.Body.ShopId, e.Body.CharacterId)
		}
	}
}

func handleCapacityFullEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventCapacityFullBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventCapacityFullBody]) {
		if e.Type != merchant2.StatusEventCapacityFull {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Shop [%s] is full, character [%d] cannot enter.", e.Body.ShopId, e.CharacterId)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)(interactionpkt.CharacterInteractionEnterResultErrorBody(interactionpkt.CharacterInteractionEnterErrorModeFull)))
	}
}

func handlePurchaseFailedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventPurchaseFailedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventPurchaseFailedBody]) {
		if e.Type != merchant2.StatusEventPurchaseFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Purchase failed in shop [%s] for character [%d]. reason [%s].", e.Body.ShopId, e.CharacterId, e.Body.Reason)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)(interactionpkt.CharacterInteractionEnterResultErrorBody(interactionpkt.CharacterInteractionEnterErrorModeUnable)))
	}
}

func handleFrederickNotificationEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventFrederickNotificationBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventFrederickNotificationBody]) {
		if e.Type != merchant2.StatusEventFrederickNotification {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Frederick notification for character [%d]. daysSinceStorage [%d].", e.CharacterId, e.Body.DaysSinceStorage)

		msg := fmt.Sprintf("Your hired merchant items have been stored for %d day(s). Please retrieve them from Fredrick.", e.Body.DaysSinceStorage)
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(merchantpkt.HiredMerchantOperationWriter)(merchantpkt.HiredMerchantOperationFreeFormNoticeBody(msg)))
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

		l.Debugf("Message sent in shop [%s] by character [%d] slot [%d].", e.Body.ShopId, e.Body.CharacterId, e.Body.Slot)

		broadcastToShopViewers(l, ctx, sc, wp, e.Body.ShopId, func(characterIds []uint32) {
			sp := session.NewProcessor(l, ctx)
			announce := session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)(interactionpkt.CharacterInteractionChatBody(e.Body.Slot, e.Body.Content))
			for _, cid := range characterIds {
				_ = sp.IfPresentByCharacterId(sc.Channel())(cid, announce)
			}
		})
	}
}

func handleListingPurchasedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.ListingEvent[merchant2.ListingEventPurchasedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.ListingEvent[merchant2.ListingEventPurchasedBody]) {
		if e.Type != merchant2.ListingEventPurchased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Listing [%d] purchased in shop [%s] by character [%d]. bundleCount [%d], remaining [%d].", e.Body.ListingIndex, e.ShopId, e.Body.BuyerCharacterId, e.Body.BundleCount, e.Body.BundlesRemaining)
	}
}

func broadcastToShopViewers(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, shopId string, fn func(characterIds []uint32)) {
	mp := merchant.NewProcessor(l, ctx)
	shop, err := mp.GetShop(shopId)
	if err != nil {
		l.WithError(err).Errorf("Unable to get shop [%s] for broadcast.", shopId)
		return
	}

	characterIds := []uint32{shop.CharacterId()}
	characterIds = append(characterIds, shop.Visitors()...)
	fn(characterIds)
}
