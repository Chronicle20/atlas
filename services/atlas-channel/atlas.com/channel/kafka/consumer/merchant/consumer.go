package merchant

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	merchant2 "atlas-channel/kafka/message/merchant"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/merchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	mapId "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("merchant_status_event")(merchant2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("merchant_listing_event")(merchant2.EnvListingEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(merchant2.EnvStatusEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShopOpenedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleShopSetupEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleShopClosedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleVisitorEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleCapacityFullEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleShopCreateFailedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handlePurchaseFailedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleFrederickNotificationEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleMessageSentEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleMaintenanceEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				t, _ = topic.EnvProvider(l)(merchant2.EnvListingEventTopic)()
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleListingPurchasedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
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

		if e.Body.ShopType == merchant.HiredMerchantShopType {
			// Hired merchant renders as a standalone CEmployeePool NPC that persists
			// independent of the owner avatar (D1), so broadcast the employee spawn
			// to everyone in the map rather than a box on the owner.
			shop, err := merchant.NewProcessor(l, ctx).GetShop(e.Body.ShopId)
			if err != nil {
				l.WithError(err).Errorf("Unable to load hired-merchant shop [%s] for spawn.", e.Body.ShopId)
			} else {
				spawn := merchant.ToEmployeeSpawn(shop, resolveOwnerName(l, ctx, shop.CharacterId()))
				if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(merchantcb.MerchantEmployeeSpawnWriter)(spawn.Encode)); err != nil {
					l.WithError(err).Errorf("Unable to spawn hired merchant [%s] in map [%d] instance [%s].", e.Body.ShopId, e.Body.MapId, e.Body.InstanceId)
				}
			}
		} else {
			// Personal store: the box attaches to the owner's own avatar.
			mr := &interactionpkt.MiniRoomBase{
				MiniRoomTypeVal: interactionpkt.PersonalShopMiniRoomType,
				Title:           e.Body.Title,
				CapacityVal:     4,
				OwnerId:         e.CharacterId,
				VisitorCount:    0,
				VisitorList:     []interactionpkt.MiniRoomVisitor{},
			}
			if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(e.CharacterId))); err != nil {
				l.WithError(err).Errorf("Unable to spawn personal store [%s] for characters in map [%d] instance [%s].", e.Body.ShopId, e.Body.MapId, e.Body.InstanceId)
			}
		}

		// No room re-send to the owner here: the owner has held the shop dialog
		// since SHOP_SETUP (create); going live only broadcasts the map
		// box/employee spawn. Re-sending ENTER_RESULT would re-create the
		// owner's dialog at go-live.
	}
}

// handleShopSetupEvent drops the owner of a freshly-created (Draft) shop into
// the shop UI so they can stock it before the formal open. Unlike SHOP_OPENED it
// does NOT spawn the mini-room box on the map — that happens at go-live
// (SHOP_OPENED), avoiding a double spawn.
func handleShopSetupEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventShopOpenedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventShopOpenedBody]) {
		if e.Type != merchant2.StatusEventShopSetup {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.Body.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Shop [%s] created by character [%d] entering setup in map [%d] instance [%s].", e.Body.ShopId, e.CharacterId, e.Body.MapId, e.Body.InstanceId)

		room, err := buildShopRoomFirstTime(l, ctx, e.Body.ShopId, e.CharacterId, true)
		if err != nil {
			l.WithError(err).Errorf("Unable to build setup room for shop [%s].", e.Body.ShopId)
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultSuccessBody(room)))
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

		// Fetch shop to get field data for map-wide despawn broadcast.
		mp := merchant.NewProcessor(l, ctx)
		shop, err := mp.GetShop(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop [%s] for close broadcast.", e.Body.ShopId)
			return
		}

		f := field.NewBuilder(shop.WorldId(), shop.ChannelId(), mapId.Id(shop.MapId())).SetInstance(shop.InstanceId()).Build()
		if shop.ShopType() == merchant.HiredMerchantShopType {
			destroy := merchantcb.NewEmployeeDestroy(e.CharacterId)
			if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(merchantcb.MerchantEmployeeDestroyWriter)(destroy.Encode)); err != nil {
				l.WithError(err).Errorf("Unable to broadcast employee despawn for shop [%s].", e.Body.ShopId)
			}
		} else {
			mr := &interactionpkt.MiniRoomBase{}
			if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Despawn(e.CharacterId))); err != nil {
				l.WithError(err).Errorf("Unable to broadcast despawn for shop [%s].", e.Body.ShopId)
			}
		}
	}
}

func handleVisitorEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventVisitorBody]) {
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		sp := session.NewProcessor(l, ctx)

		announce := session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)

		switch e.Type {
		case merchant2.StatusEventVisitorEntered:
			l.Debugf("Visitor [%d] entered shop [%s] at slot [%d].", e.Body.CharacterId, e.Body.ShopId, e.Body.Slot)

			// A completed visit consumes any pending owl-warp entry (task-127).
			shopscanner.GetRegistry().RemovePending(t, e.Body.CharacterId)

			// Refresh the field balloon's visitor count for onlookers in the map.
			broadcastEmployeeBalloonUpdate(l, ctx, wp, e.Body.ShopId)

			// Send full shop interior to the entering visitor.
			room, err := buildShopRoom(l, ctx, e.Body.ShopId, e.Body.CharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to build room for shop [%s] for visitor [%d].", e.Body.ShopId, e.Body.CharacterId)
			} else {
				_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, announce(interactioncb.CharacterInteractionEnterResultSuccessBody(room)))
			}

			// Broadcast ENTER with avatar to existing viewers so they see the new visitor.
			broadcastToShopViewers(l, ctx, sc, wp, e.Body.ShopId, func(characterIds []uint32) {
				cp := character.NewProcessor(l, ctx)
				c, err := cp.GetById(cp.InventoryDecorator)(e.Body.CharacterId)
				if err != nil {
					l.WithError(err).Warnf("Unable to resolve entering visitor [%d].", e.Body.CharacterId)
					return
				}
				avatar := model.NewFromCharacter(c, false)
				visitor := interactionpkt.NewBaseVisitor(e.Body.Slot, avatar, c.Name())
				for _, cid := range characterIds {
					if cid == e.Body.CharacterId {
						continue
					}
					_ = sp.IfPresentByCharacterId(sc.Channel())(cid, announce(interactioncb.CharacterInteractionEnterBody(visitor)))
				}
			})
		case merchant2.StatusEventVisitorExited:
			l.Debugf("Visitor [%d] exited shop [%s] from slot [%d].", e.Body.CharacterId, e.Body.ShopId, e.Body.Slot)

			// Refresh the field balloon's visitor count for onlookers in the map.
			broadcastEmployeeBalloonUpdate(l, ctx, wp, e.Body.ShopId)

			// Send LEAVE to the exiting visitor (closes their room UI).
			_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, announce(interactioncb.CharacterInteractionLeaveBody(e.Body.Slot, 0)))

			// Broadcast LEAVE to owner + remaining visitors (removes avatar from slot).
			broadcastToShopViewers(l, ctx, sc, wp, e.Body.ShopId, func(characterIds []uint32) {
				for _, cid := range characterIds {
					_ = sp.IfPresentByCharacterId(sc.Channel())(cid, announce(interactioncb.CharacterInteractionLeaveBody(e.Body.Slot, 0)))
				}
			})
		case merchant2.StatusEventVisitorEjected:
			l.Debugf("Visitor [%d] ejected from shop [%s] from slot [%d].", e.Body.CharacterId, e.Body.ShopId, e.Body.Slot)

			// Refresh the field balloon's visitor count for onlookers in the map.
			broadcastEmployeeBalloonUpdate(l, ctx, wp, e.Body.ShopId)

			// Send LEAVE to the ejected visitor (closes their room UI).
			_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, announce(interactioncb.CharacterInteractionLeaveBody(e.Body.Slot, 0)))
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

			// Send full shop interior to the owner entering maintenance.
			room, err := buildShopRoom(l, ctx, e.Body.ShopId, e.Body.CharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to build room for shop [%s] for maintenance.", e.Body.ShopId)
			} else {
				_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultSuccessBody(room)))
			}
		case merchant2.StatusEventMaintenanceExited:
			l.Debugf("Maintenance exited for shop [%s] by character [%d].", e.Body.ShopId, e.Body.CharacterId)

			mp := merchant.NewProcessor(l, ctx)
			shop, err := mp.GetShop(e.Body.ShopId)
			if err != nil {
				l.WithError(err).Errorf("Unable to get shop [%s] for maintenance exit.", e.Body.ShopId)
				return
			}

			sp := session.NewProcessor(l, ctx)
			if shop.ShopType() == merchant.HiredMerchantShopType {
				// Hired merchant: close management UI, shop continues autonomously.
				_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionLeaveBody(0, 0)))
			} else {
				// Personal shop: refresh room with updated listings.
				room, err := buildShopRoom(l, ctx, e.Body.ShopId, e.Body.CharacterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to build room for shop [%s] after maintenance exit.", e.Body.ShopId)
					return
				}
				_ = sp.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultSuccessBody(room)))
			}
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

		reg := shopscanner.GetRegistry()
		if _, ok := reg.GetPending(t, e.CharacterId); ok {
			// Owl warp arrival hit a full shop: answer with the faithful
			// SHOP_LINK code 2 instead of the mini-room error (task-127).
			reg.RemovePending(t, e.CharacterId)
			_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(merchantcb.ShopLinkResultWriter)(writer.ShopLinkResultBody(merchantpkt.ShopLinkResultCodeFull)))
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(interactioncb.CharacterInteractionEnterErrorModeFull)))
	}
}

func handleShopCreateFailedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event merchant2.StatusEvent[merchant2.StatusEventShopCreateFailedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.StatusEvent[merchant2.StatusEventShopCreateFailedBody]) {
		if e.Type != merchant2.StatusEventShopCreateFailed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.Body.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Store placement failed for character [%d]. reason [%s].", e.CharacterId, e.Body.Reason)

		mode := shopCreateFailureMode(e.Body.Reason)
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(mode)))
	}
}

// shopCreateFailureMode maps a merchant create-failure reason to the client's
// mini-room error mode.
func shopCreateFailureMode(reason string) interactioncb.CharacterInteractionEnterErrorMode {
	switch reason {
	case merchant2.ShopCreateFailReasonTooCloseToPortal:
		return interactioncb.CharacterInteractionEnterErrorModeCannotOpenStoreNearPortal
	case merchant2.ShopCreateFailReasonTooCloseToShop:
		return interactioncb.CharacterInteractionEnterErrorModeCannotOpenMiniRoomHere
	case merchant2.ShopCreateFailReasonNotFreeMarket:
		return interactioncb.CharacterInteractionEnterErrorModeMustBeInFreeMarket
	default:
		return interactioncb.CharacterInteractionEnterErrorModeUnable
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(interactioncb.CharacterInteractionEnterErrorModeUnable)))
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
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(merchantcb.HiredMerchantOperationWriter)(merchantpkt.HiredMerchantOperationFreeFormNoticeBody(msg)))
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
			announce := session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionChatBody(e.Body.Slot, e.Body.Content))
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

		mp := merchant.NewProcessor(l, ctx)
		shop, err := mp.GetShop(e.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop [%s] for purchase update.", e.ShopId)
			return
		}

		items := buildShopItems(l, shop.Listings())

		sp := session.NewProcessor(l, ctx)
		characterIds := []uint32{shop.CharacterId()}
		characterIds = append(characterIds, shop.Visitors()...)
		for _, cid := range characterIds {
			meso := uint32(0)
			if cid == shop.CharacterId() {
				meso = shop.MesoBalance()
			}
			_ = sp.IfPresentByCharacterId(sc.Channel())(cid, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionUpdateMerchantBody(meso, items)))
		}
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

// resolveOwnerName looks up a character's name for the hired-merchant employee
// nametag, returning "" (a valid empty nametag) if the lookup fails.
func resolveOwnerName(l logrus.FieldLogger, ctx context.Context, characterId uint32) string {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve owner name for character [%d].", characterId)
		return ""
	}
	return c.Name()
}

// broadcastEmployeeBalloonUpdate refreshes a hired merchant's field balloon (e.g.
// its visitor count) for everyone in the map via CEmployeePool::OnEmployeeMiniRoomBalloon.
// No-op for personal stores (which have no employee balloon).
func broadcastEmployeeBalloonUpdate(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, shopId string) {
	shop, err := merchant.NewProcessor(l, ctx).GetShop(shopId)
	if err != nil {
		l.WithError(err).Warnf("Unable to load shop [%s] for employee balloon refresh.", shopId)
		return
	}
	if shop.ShopType() != merchant.HiredMerchantShopType {
		return
	}
	f := field.NewBuilder(shop.WorldId(), shop.ChannelId(), mapId.Id(shop.MapId())).SetInstance(shop.InstanceId()).Build()
	update := merchant.ToEmployeeUpdate(shop)
	if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(merchantcb.MerchantEmployeeUpdateWriter)(update.Encode)); err != nil {
		l.WithError(err).Warnf("Unable to broadcast employee balloon update for shop [%s].", shopId)
	}
}

// buildShopRoom fetches shop data and resolves character info to build the appropriate room packet.
func buildShopRoom(l logrus.FieldLogger, ctx context.Context, shopId string, viewerCharacterId uint32) (interactionpkt.Room, error) {
	return buildShopRoomFirstTime(l, ctx, shopId, viewerCharacterId, false)
}

// buildShopRoomFirstTime builds the enter-result room for viewerCharacterId.
// firstTime marks the hired-merchant owner's creation-time view (the client's
// Decode1 @0x518b0a branches the owner UI on it); pass true only from the
// SHOP_SETUP handler.
func buildShopRoomFirstTime(l logrus.FieldLogger, ctx context.Context, shopId string, viewerCharacterId uint32, firstTime bool) (interactionpkt.Room, error) {
	mp := merchant.NewProcessor(l, ctx)
	shop, err := mp.GetShop(shopId)
	if err != nil {
		return interactionpkt.Room{}, err
	}

	cp := character.NewProcessor(l, ctx)

	items := buildShopItems(l, shop.Listings())

	position, err := viewerPosition(shop, viewerCharacterId)
	if err != nil {
		return interactionpkt.Room{}, err
	}

	// CharacterShop (1) = PersonalShop, HiredMerchant (2) = MerchantShop.
	if shop.ShopType() == merchant.HiredMerchantShopType {
		return buildMerchantShopRoom(l, ctx, cp, shop, position, firstTime, items)
	}
	return buildPersonalShopRoom(l, ctx, cp, shop, position, items)
}

// viewerPosition resolves the recipient's position in the room: 0 for the
// owner, the 1-indexed visitor slot otherwise (shop.Visitors() is the
// insertion-ordered visitor registry — the same ordering the merchant service
// uses for VISITOR_* event slots).
func viewerPosition(shop merchant.Model, viewerCharacterId uint32) (byte, error) {
	if viewerCharacterId == shop.CharacterId() {
		return 0, nil
	}
	for i, visitorId := range shop.Visitors() {
		if visitorId == viewerCharacterId {
			return byte(i + 1), nil
		}
	}
	return 0, fmt.Errorf("character [%d] is neither owner nor visitor of shop [%s]", viewerCharacterId, shop.Id())
}

func buildMerchantShopRoom(l logrus.FieldLogger, ctx context.Context, cp character.Processor, shop merchant.Model, position byte, firstTime bool, items []interactionpkt.RoomShopItem) (interactionpkt.Room, error) {
	// Owner is represented as MerchantVisitor (NPC sprite with item ID).
	ownerVisitor := interactionpkt.NewMerchantVisitor(shop.PermitItemId(), shop.Title())

	visitors := []interactionpkt.Visitor{ownerVisitor}
	for i, visitorId := range shop.Visitors() {
		c, err := cp.GetById(cp.InventoryDecorator)(visitorId)
		if err != nil {
			l.WithError(err).Warnf("Unable to resolve visitor [%d].", visitorId)
			continue
		}
		avatar := model.NewFromCharacter(c, false)
		visitors = append(visitors, interactionpkt.NewBaseVisitor(byte(i+1), avatar, c.Name()))
	}

	// Owner-only message list (position 0): replay the persisted chat log.
	// Sender names are embedded ("Name : text") because the live room's slot
	// assignments say nothing about who sent a message while the merchant ran
	// unattended. The trailing byte is the message type; 0 renders as a
	// normal chat line.
	var messages []interactionpkt.RoomMessage
	if position == 0 {
		names := map[uint32]string{}
		for _, mm := range shop.Messages() {
			name, ok := names[mm.CharacterId()]
			if !ok {
				if c, err := cp.GetById()(mm.CharacterId()); err == nil {
					name = c.Name()
				}
				names[mm.CharacterId()] = name
			}
			content := mm.Content()
			if name != "" {
				content = name + " : " + content
			}
			messages = append(messages, interactionpkt.RoomMessage{Message: content, Slot: 0})
		}
	}

	ownerChar, err := cp.GetById()(shop.CharacterId())
	if err != nil {
		return interactionpkt.Room{}, fmt.Errorf("unable to resolve owner [%d]: %w", shop.CharacterId(), err)
	}

	// position 0 (owner) additionally carries the open-time/first-time/ledger
	// block (CEntrustedShopDlg::OnEnterResult zero-branch @0x518a7e). Open time
	// is approximated as minutes since creation (retail counts from OPEN; the
	// Draft window is normally minutes). No per-sale ledger is tracked
	// server-side, so the ledger is empty with the accrued balance as total.
	room := interactionpkt.NewMerchantShopRoom(position, visitors, messages, ownerChar.Name(), shop.Title(), 16, shop.MesoBalance(), items)
	if position == 0 {
		room = room.SetOwnerLedger(minutesSince(shop.CreatedAt()), firstTime, nil, shop.MesoBalance())
	}
	return room, nil
}

// minutesSince converts elapsed wall time to the uint16 minute count the
// merchant owner view displays, saturating instead of wrapping.
func minutesSince(t time.Time) uint16 {
	if t.IsZero() {
		return 0
	}
	m := int64(time.Since(t) / time.Minute)
	if m < 0 {
		return 0
	}
	if m > 65535 {
		return 65535
	}
	return uint16(m)
}

func buildPersonalShopRoom(l logrus.FieldLogger, ctx context.Context, cp character.Processor, shop merchant.Model, position byte, items []interactionpkt.RoomShopItem) (interactionpkt.Room, error) {
	// Owner is slot 0 as a regular visitor with avatar.
	ownerChar, err := cp.GetById(cp.InventoryDecorator)(shop.CharacterId())
	if err != nil {
		return interactionpkt.Room{}, fmt.Errorf("unable to resolve owner [%d]: %w", shop.CharacterId(), err)
	}
	ownerAvatar := model.NewFromCharacter(ownerChar, false)
	visitors := []interactionpkt.Visitor{interactionpkt.NewBaseVisitor(0, ownerAvatar, ownerChar.Name())}

	for i, visitorId := range shop.Visitors() {
		c, err := cp.GetById(cp.InventoryDecorator)(visitorId)
		if err != nil {
			l.WithError(err).Warnf("Unable to resolve visitor [%d].", visitorId)
			continue
		}
		avatar := model.NewFromCharacter(c, false)
		visitors = append(visitors, interactionpkt.NewBaseVisitor(byte(i+1), avatar, c.Name()))
	}

	// position is the recipient's slot in the room (0 = owner): the client
	// branches owner/visitor view on it (CPersonalShopDlg::OnEnterResult
	// @0x6fc528 — ZERO = owner add-item management UI).
	return interactionpkt.NewPersonalShopRoom(position, visitors, shop.Title(), 16, items), nil
}

// buildShopItems converts listing models to packet RoomShopItems.
func buildShopItems(l logrus.FieldLogger, listings []merchant.ListingModel) []interactionpkt.RoomShopItem {
	items := make([]interactionpkt.RoomShopItem, 0, len(listings))
	for _, listing := range listings {
		asset := assetFromSnapshot(listing.ItemId(), listing.ItemSnapshot())
		items = append(items, interactionpkt.RoomShopItem{
			PerBundle: listing.BundleSize(),
			Quantity:  listing.BundlesRemaining(),
			Price:     listing.PricePerBundle(),
			Asset:     asset,
		})
	}
	return items
}

// assetFromSnapshot converts a typed AssetData into a packet model Asset.
func assetFromSnapshot(templateId uint32, s merchant.AssetData) packetmodel.Asset {
	base := packetmodel.NewAsset(true, 0, templateId, s.Expiration)

	invType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return base
	}

	if invType == inventory.TypeValueEquip {
		base = base.SetEquipmentStats(
			s.Strength, s.Dexterity, s.Intelligence, s.Luck,
			s.Hp, s.Mp,
			s.WeaponAttack, s.MagicAttack, s.WeaponDefense, s.MagicDefense,
			s.Accuracy, s.Avoidability, s.Hands, s.Speed, s.Jump,
		)
		base = base.SetEquipmentMeta(s.Slots, s.LevelType, s.Level, s.Experience, s.HammersApplied, s.Flag)
		if s.CashId != 0 {
			base = base.SetCashId(s.CashId)
		}
	} else {
		base = base.SetStackableInfo(s.Quantity, s.Flag, s.Rechargeable)
	}

	return base
}
