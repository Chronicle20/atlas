package cashshop

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/wallet"
	consumer2 "atlas-channel/kafka/consumer"
	cashshop2 "atlas-channel/kafka/message/cashshop"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_shop_status_event")(cashshop2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(cashshop2.EnvEventTopicStatus)()
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

			// Announce the purchase success to the character session
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
		return
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
		return
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

		// Use the generic error handler
		op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopInventoryCapacityIncreaseFailedBody(e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
		return
	}
}
