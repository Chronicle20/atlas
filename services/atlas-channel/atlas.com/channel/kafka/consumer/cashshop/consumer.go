package cashshop

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/wallet"
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	cashshop2 "atlas-channel/kafka/message/cashshop"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_shop_status_event")(cashshop2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(cashshop2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventInventoryCapacityIncreased(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventPurchase(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCashItemMovedToInventory(sc, wp))))
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopInventoryCapacityIncreaseSuccessBody(l)(e.Body.InventoryType, e.Body.Capacity))(s)
			if err != nil {
				return err
			}
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				w = wallet.Model{}
			}
			err = session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)(writer.CashShopCashQueryResultBody(t)(w.Credit(), w.Points(), w.Prepaid()))(s)
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			// Retrieve the asset that was purchased
			a, err := asset.NewProcessor(l, ctx).GetById(s.AccountId(), e.Body.CompartmentId, e.Body.AssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve asset [%s] for character [%d].", e.Body.AssetId, e.CharacterId)
				return err
			}

			// Announce the purchase success to the character session
			err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashInventoryPurchaseSuccessBody(l)(s.AccountId(), e.CharacterId, a))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop purchase success to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
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
		op := session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopInventoryCapacityIncreaseFailedBody(l)(e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, op)
		return
	}
}

func handleStatusEventCashItemMovedToInventory(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.CashItemMovedToInventoryEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.CashItemMovedToInventoryEventBody]) {
		if e.Type != cashshop2.StatusEventTypeCashItemMovedToInventory {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			// Retrieve the character and decorate with inventory
			c, err := character.NewProcessor(l, ctx).GetById(character.NewProcessor(l, ctx).InventoryDecorator)(e.CharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character [%d].", e.CharacterId)
				return err
			}

			comp, ok := c.Inventory().CompartmentById(e.Body.CompartmentId)
			if !ok {
				l.Errorf("Unable to retrieve compartment [%s] for character [%d].", e.Body.CompartmentId, e.CharacterId)
				return nil
			}
			as, ok := comp.FindBySlot(e.Body.Slot)
			if !ok {
				l.Errorf("Unable to retrieve asset in slot [%d] of compartment [%s] for character [%d].", e.Body.Slot, e.Body.CompartmentId, e.CharacterId)
				return nil
			}

			// Retrieve the asset from the character's inventory
			// Announce to the character that the cash item has been moved to their inventory
			err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashItemMovedToInventoryBody(l, t)(*as))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash item moved to inventory to character [%d].", e.CharacterId)
				return err
			}
			l.Infof("Cash item moved to inventory for character [%d], compartment [%s], slot [%d]", e.CharacterId, e.Body.CompartmentId, e.Body.Slot)

			return nil
		})
		return
	}
}
