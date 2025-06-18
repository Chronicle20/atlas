package cashshop

import (
	asset2 "atlas-channel/asset"
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/cashshop/wallet"
	compartment3 "atlas-channel/compartment"
	consumer2 "atlas-channel/kafka/consumer"
	cashshop2 "atlas-channel/kafka/message/cashshop"
	compartment2 "atlas-channel/kafka/message/compartment"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
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
			rf(consumer2.NewConfig(l)("compartment_transfer_status_event")(compartment2.EnvEventTopicCompartmentTransferStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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

				// Register the new handler for compartment transfer status events
				t, _ = topic.EnvProvider(l)(compartment2.EnvEventTopicCompartmentTransferStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentTransferStatusEventCompleted(sc, wp))))
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

func handleCompartmentTransferStatusEventCompleted(sc server.Model, wp writer.Producer) message.Handler[compartment2.TransferStatusEvent[compartment2.TransferStatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment2.TransferStatusEvent[compartment2.TransferStatusEventCompletedBody]) {
		if e.Type != compartment2.StatusEventTypeCompleted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			if e.Body.InventoryType == compartment2.InventoryTypeCharacter {
				comp, err := compartment3.NewProcessor(l, ctx).GetByType(e.CharacterId, inventory.Type(e.Body.CompartmentType))
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve compartment [%s] for character [%d].", e.Body.CompartmentId, e.CharacterId)
					return err
				}

				// Find the asset in the compartment
				var as *asset2.Model[any]
				for _, a := range comp.Assets() {
					if a.ReferenceId() == e.Body.AssetId {
						as = &a
						break
					}
				}

				if as == nil {
					l.Errorf("Unable to retrieve asset [%d] in compartment [%s] for character [%d].", e.Body.AssetId, e.Body.CompartmentId, e.CharacterId)
					return nil
				}

				// Announce to the character that the cash item has been moved to their inventory
				err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashItemMovedToInventoryBody(l, t)(*as))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce cash item moved to inventory to character [%d].", e.CharacterId)
					return err
				}
				l.Infof("Cash item moved to inventory for character [%d], compartment [%s], asset [%d]", e.CharacterId, e.Body.CompartmentId, e.Body.AssetId)

				return nil
			} else if e.Body.InventoryType == compartment2.InventoryTypeCashShop {
				// Retrieve the asset that was purchased
				comp, err := compartment.NewProcessor(l, ctx).GetByAccountIdAndType(s.AccountId(), compartment.CompartmentType(e.Body.CompartmentType))
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve compartment [%s] for character [%d].", e.Body.CompartmentId, e.CharacterId)
					return err
				}

				// Find the asset in the compartment
				var as *asset.Model
				for _, a := range comp.Assets() {
					if a.Item().Id() == e.Body.AssetId {
						as = &a
						break
					}
				}

				if as == nil {
					l.Errorf("Unable to retrieve asset [%d] in compartment [%s] for character [%d].", e.Body.AssetId, e.Body.CompartmentId, e.CharacterId)
					return nil
				}

				// Announce to the character that the cash item has been moved to their cash inventory
				err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashItemMovedToCashInventoryBody(l, t)(s.AccountId(), s.CharacterId(), *as))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce cash item moved to cash inventory to character [%d].", e.CharacterId)
					return err
				}
				l.Infof("Cash item moved to cash inventory for character [%d], compartment [%s], asset [%d]", e.CharacterId, e.Body.CompartmentId, e.Body.AssetId)

				return nil
			}
			l.Warnf("Unknown inventory type [%s] for compartment transfer status event.", e.Body.InventoryType)
			return nil
		})
		return
	}
}
