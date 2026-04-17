package merchant

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	merchant2 "atlas-channel/kafka/message/merchant"
	_map "atlas-channel/map"
	"atlas-channel/merchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

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

		miniRoomType := interactionpkt.MerchantShopMiniRoomType
		if e.Body.ShopType == 1 {
			miniRoomType = interactionpkt.PersonalShopMiniRoomType
		}

		mr := &interactionpkt.MiniRoomBase{
			MiniRoomTypeVal: miniRoomType,
			Title:           e.Body.Title,
			CapacityVal:     4,
			OwnerId:         e.CharacterId,
			VisitorCount:    0,
			VisitorList:     []interactionpkt.MiniRoomVisitor{},
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(e.CharacterId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn merchant shop [%s] for characters in map [%d] instance [%s].", e.Body.ShopId, e.Body.MapId, e.Body.InstanceId)
		}

		// Send room enter packet to the owner so they enter the shop editing interface.
		room, err := buildShopRoom(l, ctx, e.Body.ShopId, e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to build room for shop [%s].", e.Body.ShopId)
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
		mr := &interactionpkt.MiniRoomBase{}
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Despawn(e.CharacterId))); err != nil {
			l.WithError(err).Errorf("Unable to broadcast despawn for shop [%s].", e.Body.ShopId)
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
			if shop.ShopType() == 2 {
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(interactioncb.CharacterInteractionEnterErrorModeFull)))
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

// buildShopRoom fetches shop data and resolves character info to build the appropriate room packet.
func buildShopRoom(l logrus.FieldLogger, ctx context.Context, shopId string, viewerCharacterId uint32) (interactionpkt.Room, error) {
	mp := merchant.NewProcessor(l, ctx)
	shop, err := mp.GetShop(shopId)
	if err != nil {
		return interactionpkt.Room{}, err
	}

	cp := character.NewProcessor(l, ctx)

	items := buildShopItems(l, shop.Listings())

	// CharacterShop (1) = PersonalShop, HiredMerchant (2) = MerchantShop.
	if shop.ShopType() == 2 {
		return buildMerchantShopRoom(l, ctx, cp, shop, viewerCharacterId, items)
	}
	return buildPersonalShopRoom(l, ctx, cp, shop, items)
}

func buildMerchantShopRoom(l logrus.FieldLogger, ctx context.Context, cp character.Processor, shop merchant.Model, viewerCharacterId uint32, items []interactionpkt.RoomShopItem) (interactionpkt.Room, error) {
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

	// Messages are only visible to the owner.
	var messages []interactionpkt.RoomMessage
	if viewerCharacterId == shop.CharacterId() {
		// No stored messages currently; placeholder for future implementation.
	}

	ownerChar, err := cp.GetById()(shop.CharacterId())
	if err != nil {
		return interactionpkt.Room{}, fmt.Errorf("unable to resolve owner [%d]: %w", shop.CharacterId(), err)
	}

	return interactionpkt.NewMerchantShopRoom(visitors, messages, ownerChar.Name(), 16, shop.MesoBalance(), items), nil
}

func buildPersonalShopRoom(l logrus.FieldLogger, ctx context.Context, cp character.Processor, shop merchant.Model, items []interactionpkt.RoomShopItem) (interactionpkt.Room, error) {
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

	return interactionpkt.NewPersonalShopRoom(visitors, shop.Title(), 16, items), nil
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
