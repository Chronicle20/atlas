package compartment

import (
	"atlas-channel/cashshop/inventory/asset"
	consumer2 "atlas-channel/kafka/consumer"
	cashshopCompartment "atlas-channel/kafka/message/cashshop/compartment"
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
	cashpkt "github.com/Chronicle20/atlas-packet/cash"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_compartment_status_event")(cashshopCompartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(cashshopCompartment.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// handleAcceptedEvent handles when cash-shop compartment accepts an item (item moved FROM character TO cash-shop)
func handleAcceptedEvent(sc server.Model, wp writer.Producer) message.Handler[cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventAcceptedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventAcceptedBody]) {
		if e.Type != cashshopCompartment.StatusEventTypeAccepted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Cash-shop compartment accepted item. CharacterId: [%d], AssetId: [%d].", e.CharacterId, e.Body.AssetId)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// Fetch the cash-shop asset that was just created
			a, err := asset.NewProcessor(l, ctx).GetById(e.AccountId, e.CompartmentId, e.Body.AssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash-shop asset with ID [%d] for character [%d].", e.Body.AssetId, e.CharacterId)
				return err
			}

			// Notify the client that the item was moved to the cash inventory
			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   e.AccountId,
				CharacterId: e.CharacterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(a.Expiration()),
			}
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopCashItemMovedToCashInventoryBody(item))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash item moved to cash inventory for character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// handleReleasedEvent handles when cash-shop compartment releases an item (item moved FROM cash-shop TO character)
// Note: The client notification (CashShopCashItemMovedToInventory) is handled by the asset ACCEPTED event
// consumer, which fires after the item has actually been accepted into the character's inventory.
func handleReleasedEvent(sc server.Model, wp writer.Producer) message.Handler[cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventReleasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventReleasedBody]) {
		if e.Type != cashshopCompartment.StatusEventTypeReleased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Cash-shop compartment released item. CharacterId: [%d], CashId: [%d], TemplateId: [%d].", e.CharacterId, e.Body.CashId, e.Body.TemplateId)
	}
}
