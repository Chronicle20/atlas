package cashshop

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/wallet"
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	cashshop2 "atlas-channel/kafka/message/cashshop"
	"atlas-channel/listener"
	"atlas-channel/pendingchange"
	"atlas-channel/ring"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// resolvePendingChange resolves a purchase-outcome event's TransactionId to
// its correlated PENDING name-change / world-transfer record, if any. The
// zero UUID means "no correlation" (every non-name-change/world-transfer
// purchase, per task-227 task 37) and is rejected up front without a remote
// call. A non-zero id that does not match any of the character's PENDING
// records (already resolved, or a stale/foreign id) is also "no
// correlation" -- the caller falls back to its pre-existing behavior.
func resolvePendingChange(l logrus.FieldLogger, ctx context.Context, characterId uint32, transactionId uuid.UUID) (pendingchange.RestModel, bool) {
	if transactionId == uuid.Nil {
		return pendingchange.RestModel{}, false
	}
	pcs, err := pendingchange.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		l.WithError(err).Warnf("Unable to list pending changes for character [%d] while resolving transaction [%s].", characterId, transactionId)
		return pendingchange.RestModel{}, false
	}
	for _, pc := range pcs {
		if pc.Status == pendingchange.StatusPending && pc.Id == transactionId.String() {
			return pc, true
		}
	}
	return pendingchange.RestModel{}, false
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_shop_status_event")(cashshop2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var err error
				var handles []listener.HandlerHandle
				t, err = topic.EnvProvider(l)(cashshop2.EnvEventTopicStatus)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventInventoryCapacityIncreased(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventPurchase(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventSurpriseOpened(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventSurpriseFailed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCouponRedeemed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCouponFailed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLockerRebated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGiftPurchased(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventPackagePurchased(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventRingPurchased(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventEquipSlotIncreased(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventInventoryCapacityIncreased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.InventoryCapacityIncreasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.InventoryCapacityIncreasedBody]) {
		if e.Type != cashshop2.StatusEventTypeInventoryCapacityIncreased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopInventoryCapacityIncreaseSuccessBody(e.Body.InventoryType, e.Body.Capacity))(s)
			if err != nil {
				return err
			}
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				w = wallet.Model{}
			}
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashQueryResultWriter)(cashpkt.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
				return err
			}
			return nil
		})
		return
	}
}

func handleStatusEventPurchase(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.PurchaseEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.PurchaseEventBody]) {
		if e.Type != cashshop2.StatusEventTypePurchase {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// Retrieve the asset that was purchased
			a, err := asset.NewProcessor(l, ctx).GetById(s.AccountId(), e.Body.CompartmentId, e.Body.AssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve asset [%d] for character [%d].", e.Body.AssetId, e.CharacterId)
				return err
			}

			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   s.AccountId(),
				CharacterId: e.CharacterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(a.Expiration()),
			}

			// A TransactionId correlating to a PENDING name-change /
			// world-transfer record means this purchase is the paid leg of
			// that flow -- answer with the record's own DONE arm (task-227
			// task 39) instead of the generic purchase-success body, which
			// is a different mode byte the client does not expect for
			// either op. An unrelated buy carries the zero UUID (or an id
			// that resolves to nothing) and falls through unchanged.
			if pc, ok := resolvePendingChange(l, ctx, e.CharacterId, e.Body.TransactionId); ok {
				switch pc.Type {
				case pendingchange.TypeNameChange:
					if err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopNameChangeBuyDoneBody(item))(s); err != nil {
						l.WithError(err).Errorf("Unable to announce name change success to character [%d].", e.CharacterId)
						return err
					}
					return nil
				case pendingchange.TypeWorldTransfer:
					if err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopTransferWorldDoneBody(item))(s); err != nil {
						l.WithError(err).Errorf("Unable to announce world transfer success to character [%d].", e.CharacterId)
						return err
					}
					return nil
				}
			}

			// BUY_NORMAL answers on its own SUCCESS mode byte
			// (BUY_NORMAL_SUCCESS), not the generic purchase-success body --
			// see cash_shop_operation.go's BUY_NORMAL arm for why the client
			// gets no isPoints/currency to read on this op. Per
			// docs/tasks/task-183-cashshop-result-family/arm-catalog.md's
			// BUY_NORMAL_SUCCESS row, the client reads this field as
			// nPos/slotPos and passes it to
			// CCSWnd_Inventory::SetSelectedNo to select which cash-shop
			// inventory window entry becomes highlighted after the
			// purchase -- it is not unconstrained filler. SlotPos is 0
			// (selects the first entry): neither this service's nor
			// atlas-cashshop's cash-locker asset model persists a
			// slot/ordinal position, so 0 is an interim value pending real
			// slot tracking, not a derived one.
			if e.Body.Operation == cashshop2.ErrorOperationBuyNormal {
				refs := []cashpkt.PackedCashItemRef{{
					Quantity: uint16(a.Item().Quantity()),
					SlotPos:  0,
					ItemId:   int32(a.Item().TemplateId()),
				}}
				if err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopBuyNormalDoneBody(refs))(s); err != nil {
					l.WithError(err).Errorf("Unable to announce buy normal success to character [%d].", e.CharacterId)
					return err
				}
				return nil
			}

			// Announce the purchase success to the character session
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopCashInventoryPurchaseSuccessBody(item))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop purchase success to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
		return
	}
}

// handleStatusEventCouponRedeemed announces one successful coupon redemption.
//
// It keys off Body.AssetIds, never off Body.CompartmentId: a currency-only
// coupon (prepaid / NX only) awards no locker item, so atlas-cashshop emits the
// ZERO uuid as the compartment id. That is a normal success, not an error.
func handleStatusEventCouponRedeemed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]) {
		if e.Type != cashshop2.StatusEventTypeCouponRedeemed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			ap := asset.NewProcessor(l, ctx)
			items := make([]cashpkt.CashInventoryItem, 0, len(e.Body.AssetIds))
			for _, id := range e.Body.AssetIds {
				a, err := ap.GetById(s.AccountId(), e.Body.CompartmentId, id)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve coupon asset [%d] for character [%d].", id, e.CharacterId)
					return err
				}
				items = append(items, cashpkt.CashInventoryItem{
					CashId:      a.Item().CashId(),
					AccountId:   s.AccountId(),
					CharacterId: e.CharacterId,
					TemplateId:  a.Item().TemplateId(),
					CommodityId: a.CommodityId(),
					Quantity:    int16(a.Item().Quantity()),
					GiftFrom:    "",
					Expiration:  packetmodel.MsTime(a.Expiration()),
				})
			}

			// maplePoint is the DELTA this coupon awarded — the client renders
			// it inside "You have received ... using the coupon" and skips it
			// when zero. meso is 0: meso rewards are out of scope (PRD §2).
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopUseCouponDoneBody(items, int32(e.Body.MaplePoints), nil, 0))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce coupon success to character [%d].", e.CharacterId)
				return err
			}

			// Refresh the balances so an OPEN Cash Shop window updates without
			// a relog: the client reads balances from CashQueryResult, never
			// from the coupon arm.
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				return nil
			}
			if err = session.Announce(l)(ctx)(wp)(cashpkt.CashQueryResultWriter)(cashpkt.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s); err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
			}
			return nil
		})
	}
}

// handleStatusEventCouponFailed announces a coupon failure on the
// USE_COUPON_FAILED arm specifically — the generic ERROR handler below
// announces the inventory-capacity-increase arm, which is a different mode
// byte, which is why COUPON_FAILED is its own event type.
func handleStatusEventCouponFailed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.CouponFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.CouponFailedBody]) {
		if e.Type != cashshop2.StatusEventTypeCouponFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopUseCouponFailedBody(e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
	}
}

// handleStatusEventLockerRebated announces the REBATE_SUCCESS arm. Currency
// is mirrored on LockerRebatedBody for wire compatibility with
// atlas-cashshop but is deliberately not read here: CashShopRebateDoneBody
// takes only sn/amount (shop_operation_body.go:600-604), so there is nothing
// for this arm to do with it.
func handleStatusEventLockerRebated(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.LockerRebatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.LockerRebatedBody]) {
		if e.Type != cashshop2.StatusEventTypeLockerRebated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopRebateDoneBody(e.Body.CashId, e.Body.Amount))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
	}
}

// handleStatusEventGiftPurchased announces the GIFT_SUCCESS arm to the
// SENDER's session (e.CharacterId, per GiftPurchasedStatusEventProvider's
// convention of keying the status event on the sender). Recipient-side live
// refresh is deliberately out of scope: REFRESH_LOCKER (mode 162) is not
// bound in the "operations" table of any GMS seed template, so announcing it
// would resolve to the ResolveCode sentinel. The gifted asset is durable in
// the recipient's locker either way; they see it on their next locker load.
//
// GIFT_SUCCESS must be followed by a CashQueryResult announce, in that
// order: the v83 client's gift batch state machine
// (CCashShop::SendGiftsPacket) only advances to its final notice
// (SP_561/SP_562/SP_563) on CASH_QUERY_RESULT, and only resolves correctly
// if the GIFT_SUCCESS that records the confirmation has already landed.
// Mirrors handleStatusEventInventoryCapacityIncreased above.
func handleStatusEventGiftPurchased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.GiftPurchasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.GiftPurchasedBody]) {
		if e.Type != cashshop2.StatusEventTypeGiftPurchased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopGiftDoneBody(e.Body.RecipientName, int32(e.Body.TemplateId), e.Body.Quantity, int32(e.Body.Price)))(s)
			if err != nil {
				return err
			}
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				w = wallet.Model{}
			}
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashQueryResultWriter)(cashpkt.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
				return err
			}
			return nil
		})
	}
}

// handleStatusEventPackagePurchased answers task 16's REQUEST_PACKAGE_PURCHASE
// command, covering both BUY_PACKAGE and BUY_OTHER_PACKAGE (task 17). Unlike
// the COMMAND body (where RecipientCharacterId == 0 means buy-for-self), the
// STATUS body's RecipientCharacterId/RecipientName always echo a concrete
// identity -- on a buy-for-self purchase they echo the buyer's own identity
// (kafka/message/cashshop/kafka.go:372-375, PackagePurchasedBody's doc
// comment), so RecipientCharacterId is never zero on this event. The correct
// discriminator is therefore RecipientCharacterId != e.CharacterId: equal
// means buy-for-self (BUY_PACKAGE_SUCCESS, mode 154), different means gift
// (GIFT_PACKAGE_SUCCESS, mode 156, derivation.md D3b/§5). Both arms
// announce to e.CharacterId (the buyer/sender, per
// PackagePurchasedStatusEventProvider's convention -- mirroring
// handleStatusEventGiftPurchased above).
//
// The buy-for-self arm projects Body.AssetIds into cashpkt.CashInventoryItem
// records the same way handleStatusEventPurchase does
// (kafka/consumer/cashshop/consumer.go:160-169) and
// handleStatusEventCouponRedeemed does above -- atlas-cashshop deliberately
// does not build these itself (PackagePurchasedBody's own doc comment). The
// gift arm carries no item blob at all (GiftPackageDone's TRUE SHAPE, see
// shop_operation_result_gift.go) so Body.AssetIds is not read on that path.
//
// unused1/unused2 on CashShopGiftPackageDoneBody are named "unused" in the
// clientbound constructor itself (NewGiftPackageDone) -- kept zero here;
// derivation.md did not contradict that.
func handleStatusEventPackagePurchased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.PackagePurchasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.PackagePurchasedBody]) {
		if e.Type != cashshop2.StatusEventTypePackagePurchased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if e.Body.RecipientCharacterId != e.CharacterId {
				err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopGiftPackageDoneBody(e.Body.RecipientName, int32(e.Body.PackageTemplateId), 0, 0, int32(e.Body.Price)))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce package gift success to character [%d].", e.CharacterId)
					return err
				}
				return nil
			}

			ap := asset.NewProcessor(l, ctx)
			items := make([]cashpkt.CashInventoryItem, 0, len(e.Body.AssetIds))
			for _, id := range e.Body.AssetIds {
				a, err := ap.GetById(s.AccountId(), e.Body.CompartmentId, id)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve package asset [%d] for character [%d].", id, e.CharacterId)
					return err
				}
				items = append(items, cashpkt.CashInventoryItem{
					CashId:      a.Item().CashId(),
					AccountId:   s.AccountId(),
					CharacterId: e.CharacterId,
					TemplateId:  a.Item().TemplateId(),
					CommodityId: a.CommodityId(),
					Quantity:    int16(a.Item().Quantity()),
					GiftFrom:    "",
					Expiration:  packetmodel.MsTime(a.Expiration()),
				})
			}

			err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopBuyPackageDoneBody(items, 0))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce package purchase success to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// handleStatusEventRingPurchased announces the COUPLE_SUCCESS / FRIENDSHIP_SUCCESS
// arm to the BUYER's session (e.CharacterId, per RingPurchasedBody's
// convention of keying the status event on the buyer -- mirroring
// handleStatusEventGiftPurchased/handleStatusEventPackagePurchased above).
// It projects Body.AssetId into a cashpkt.CashInventoryItem the same way
// handleStatusEventPurchase does (atlas-cashshop deliberately does not
// build this itself -- RingPurchasedBody's own doc comment), then picks
// CashShopCoupleDoneBody or CashShopFriendshipDoneBody by Body.RingType.
// The partner's own half is not announced here -- there is no live session
// correlation for it on this event (see OQ-R1: the distinct-halves
// rejection branch is unimplemented for the same reason).
//
// Both halves' ring caches are invalidated (task-269 task 12) rather than
// patched: RingPurchasedBody carries no cashId for either half and no
// partner character id (its own doc comment), so there is nothing to patch
// the cache entry with -- the next Populate call (character load) re-fetches
// the real halves. The buyer is invalidated by e.CharacterId directly. The
// partner is resolved from Body.PartnerName the same way handleRingPurchase
// resolves a purchase-time partner (cash_shop_ring.go); an unresolvable
// name (partner deleted, transient atlas-character outage) fails soft --
// the buyer's own invalidation and announcement still happen.
func handleStatusEventRingPurchased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.RingPurchasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.RingPurchasedBody]) {
		if e.Type != cashshop2.StatusEventTypeRingPurchased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		ring.NewProcessor(l, ctx).Invalidate(e.CharacterId)

		if partner, perr := character.NewProcessor(l, ctx).GetByName(e.Body.PartnerName); perr == nil {
			ring.NewProcessor(l, ctx).Invalidate(partner.Id())
		} else {
			l.WithError(perr).Debugf("Unable to resolve ring purchase partner [%s] for character [%d]; partner's ring cache left untouched.", e.Body.PartnerName, e.CharacterId)
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			a, err := asset.NewProcessor(l, ctx).GetById(s.AccountId(), e.Body.CompartmentId, e.Body.AssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve ring asset [%d] for character [%d].", e.Body.AssetId, e.CharacterId)
				return err
			}

			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   s.AccountId(),
				CharacterId: e.CharacterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(a.Expiration()),
			}

			var body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
			switch e.Body.RingType {
			case cashshop2.RingTypeCouple:
				body = cashpkt.CashShopCoupleDoneBody(item, e.Body.PartnerName, int32(e.Body.TemplateId), e.Body.Quantity)
			case cashshop2.RingTypeFriendship:
				body = cashpkt.CashShopFriendshipDoneBody(item, e.Body.PartnerName, int32(e.Body.TemplateId), e.Body.Quantity)
			default:
				l.Errorf("Unrecognized ring type [%s] for character [%d]; unable to announce ring purchase.", e.Body.RingType, e.CharacterId)
				return nil
			}

			if err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(body)(s); err != nil {
				l.WithError(err).Errorf("Unable to announce ring purchase success to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// handleStatusEventEquipSlotIncreased announces the ENABLE_EQUIP_SLOT_EXT_SUCCESS
// arm for a purchased equip slot extension (task-240 task 23). There is no
// locker asset to look up here -- the purchase creates no cash inventory
// item, closer to handleStatusEventInventoryCapacityIncreased's shape than
// to handleStatusEventRingPurchased's.
//
// e.Body.SlotIndex is the Atlas CANONICAL equipped-inventory position
// (-59, pendant2) -- see EquipSlotIncreasedBody's doc comment. The
// EnableEquipSlotExtSuccess packet body's slotIndex is a distinct WIRE
// value that this derivation pins at 0 (it indexes a one-element slot
// array), so the literal 0 below -- never e.Body.SlotIndex -- is
// deliberate: passing the event field through would silently encode
// -59 as the unsigned wire value 65477.
func handleStatusEventEquipSlotIncreased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.EquipSlotIncreasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.EquipSlotIncreasedBody]) {
		if e.Type != cashshop2.StatusEventTypeEquipSlotIncreased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopEnableEquipSlotExtSuccessBody(0, e.Body.Days))(s); err != nil {
				l.WithError(err).Errorf("Unable to announce equip slot extension success to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// failureBodyForOperation routes a cash shop failure to its own arm's
// *_FAILED body builder, keyed on ErrorEventBody.Operation. default
// reproduces today's behavior byte for byte -- CashShopInventoryCapacityIncreaseFailedBody
// -- so an existing producer that never sets Operation (empty string) is
// unaffected, as is any operation value this switch does not yet recognize.
func failureBodyForOperation(operation string, reason string) packet.Encode {
	switch operation {
	case cashshop2.ErrorOperationGift:
		return cashpkt.CashShopGiftFailedBody(reason)
	case cashshop2.ErrorOperationBuyNormal:
		return cashpkt.CashShopBuyNormalFailedBody(reason)
	case cashshop2.ErrorOperationRebate:
		return cashpkt.CashShopRebateFailedBody(reason)
	case cashshop2.ErrorOperationCouple:
		return cashpkt.CashShopCoupleFailedBody(reason)
	case cashshop2.ErrorOperationFriendship:
		return cashpkt.CashShopFriendshipFailedBody(reason)
	case cashshop2.ErrorOperationBuyPackage:
		return cashpkt.CashShopBuyPackageFailedBody(reason)
	case cashshop2.ErrorOperationGiftPackage:
		return cashpkt.CashShopGiftPackageFailedBody(reason)
	case cashshop2.ErrorOperationEnableEquipSlot:
		return cashpkt.CashShopEnableEquipSlotExtFailedBody(reason)
	default:
		return cashpkt.CashShopInventoryCapacityIncreaseFailedBody(reason)
	}
}

func handleStatusEventError(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.ErrorEventBody]) {
		if e.Type != cashshop2.StatusEventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		// A TransactionId correlating to a PENDING name-change /
		// world-transfer record means the paid leg of that flow failed
		// (task-227 task 39): release the record via the existing
		// self-scoped cancel path (reason player_cancelled,
		// pending_change/processor.go:349) -- there is no committed asset
		// on this path (HasAsset() is false), so this releases the
		// reservation without minting any refund -- and answer with the
		// record's own failure arm instead of the generic capacity-increase
		// fallback below. An unrelated failure carries the zero UUID (or an
		// id that resolves to nothing) and falls through unchanged.
		if pc, ok := resolvePendingChange(l, ctx, e.CharacterId, e.Body.TransactionId); ok {
			if _, cancelErr := pendingchange.NewProcessor(l, ctx).CancelPendingChange(e.CharacterId, pc.Type); cancelErr != nil {
				l.WithError(cancelErr).Errorf("Unable to cancel pending [%s] change for character [%d] after failed purchase [%s].", pc.Type, e.CharacterId, e.Body.TransactionId)
			}

			switch pc.Type {
			case pendingchange.TypeNameChange:
				// No NAME_CHANGE_FAILED arm exists on the wire (see
				// handleBuyNameChange's doc in socket/handler/cash_shop_operation.go);
				// pink text is the same fallback the synchronous rejection
				// path uses.
				op := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody("Unable to process your name change request."))
				_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
				return
			case pendingchange.TypeWorldTransfer:
				op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopTransferWorldFailedBody(e.Body.Error))
				_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
				return
			}
		}

		// Route to the failing arm's own *_FAILED mode byte (falls back to
		// today's capacity-increase arm for an empty/unrecognized operation).
		op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(failureBodyForOperation(e.Body.Operation, e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
		return
	}
}

func handleStatusEventSurpriseOpened(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]) {
		if e.Type != cashshop2.StatusEventTypeSurpriseOpened {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			a, err := asset.NewProcessor(l, ctx).GetById(s.AccountId(), e.Body.CompartmentId, e.Body.RewardAssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve surprise reward asset [%d] for character [%d].", e.Body.RewardAssetId, e.CharacterId)
				return err
			}

			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   s.AccountId(),
				CharacterId: e.CharacterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(a.Expiration()),
			}

			// sn is the BOX's serial — the client matches it against
			// m_aCashItemInfo[i].liSN to find the row to decrement, and
			// removes that row when remain is 0. jackpot is always 0: the
			// byte only picks CashGachaponJackpot vs CashGachaponNormal sfx
			// and the pool model has no notion of a jackpot tier.
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashItemGachaponResultWriter)(
				cashpkt.CashItemGachaponSuccessBody(
					e.Body.BoxCashId,
					int32(e.Body.BoxRemaining),
					item,
					int32(e.Body.RewardTemplateId),
					byte(e.Body.RewardCount),
					0,
				))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce surprise open result to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

func handleStatusEventSurpriseFailed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]) {
		if e.Type != cashshop2.StatusEventTypeSurpriseFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		// The FAILED arm has an empty body — the client reads only the mode
		// byte, calls StringPool::GetString(<fixed id>) and shows a notice.
		// e.Body.Reason has no field to travel in and stays server-side.
		l.Infof("Surprise open failed for character [%d]: %s.", e.CharacterId, e.Body.Reason)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashItemGachaponResultWriter)(cashpkt.CashItemGachaponFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce surprise open failure to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}
