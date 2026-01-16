package compartment

import (
	charAsset "atlas-channel/asset"
	"atlas-channel/cashshop/inventory/asset"
	charCompartment "atlas-channel/compartment"
	consumer2 "atlas-channel/kafka/consumer"
	cashshopCompartment "atlas-channel/kafka/message/cashshop/compartment"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
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
			rf(consumer2.NewConfig(l)("cash_compartment_status_event")(cashshopCompartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(cashshopCompartment.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent(sc, wp))))
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

		l.Debugf("Cash-shop compartment accepted item. CharacterId: [%d], AssetId: [%s].", e.CharacterId, e.Body.AssetId)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			// Fetch the cash-shop asset that was just created
			a, err := asset.NewProcessor(l, ctx).GetById(e.AccountId, e.CompartmentId, e.Body.AssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash-shop asset with ID [%s] for character [%d].", e.Body.AssetId, e.CharacterId)
				return err
			}

			// Notify the client that the item was moved to the cash inventory
			err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashItemMovedToCashInventoryBody(l, t)(e.AccountId, e.CharacterId, a))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash item moved to cash inventory for character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// handleReleasedEvent handles when cash-shop compartment releases an item (item moved FROM cash-shop TO character)
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			// Determine the inventory type from the template ID
			invType, ok := inventory.TypeFromItemId(item.Id(e.Body.TemplateId))
			if !ok {
				l.Errorf("Unable to determine inventory type for template [%d].", e.Body.TemplateId)
				return nil
			}

			// Query the character's inventory compartment to find the asset with matching CashId
			comp, err := charCompartment.NewProcessor(l, ctx).ByCharacterIdAndTypeProvider(e.CharacterId, invType)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character [%d] inventory compartment type [%d].", e.CharacterId, invType)
				return err
			}

			// Find the asset with matching CashId
			var foundAsset *charAsset.Model[any]
			for _, a := range comp.Assets() {
				if cashId := getCashIdFromAsset(a); cashId == e.Body.CashId {
					assetCopy := a
					foundAsset = &assetCopy
					break
				}
			}

			if foundAsset == nil {
				l.Errorf("Unable to find asset with CashId [%d] in character [%d] inventory.", e.Body.CashId, e.CharacterId)
				return nil
			}

			// Notify the client that the item was moved to inventory
			err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashItemMovedToInventoryBody(l, t)(*foundAsset))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash item moved to inventory for character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

// getCashIdFromAsset extracts the CashId from an asset's reference data if present
func getCashIdFromAsset(a charAsset.Model[any]) int64 {
	rd := a.ReferenceData()
	if rd == nil {
		return 0
	}
	if crd, ok := rd.(charAsset.CashReferenceData); ok {
		return crd.CashId()
	}
	if cerd, ok := rd.(charAsset.CashEquipableReferenceData); ok {
		return cerd.CashId()
	}
	if prd, ok := rd.(charAsset.PetReferenceData); ok {
		return prd.CashId()
	}
	return 0
}
