package inventory

import (
	"atlas-consumables/consumable"
	inventory2 "atlas-consumables/inventory"
	consumer2 "atlas-consumables/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("inventory_changed_event")(EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventInventoryChanged)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryReserve)))
	}
}

func handleInventoryReserve(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemReserveBody]) {
	if e.Type != ChangedTypeReserve {
		return
	}

	if inventory.Type(e.InventoryType) != inventory.TypeValueUse {
		return
	}
	l.Debugf("Character [%d] is attempting to consume item [%d].", e.CharacterId, e.Body.ItemId)
	itemId := item.Id(e.Body.ItemId)

	if item.GetClassification(itemId) == item.Classification(200) || item.GetClassification(itemId) == item.Classification(201) || item.GetClassification(itemId) == item.Classification(202) {
		_ = consumable.ConsumeStandard(l)(ctx)(e.CharacterId, e.Body.ItemId, e.Slot, e.Body.Quantity, e.Body.TransactionId)
		return
	} else if item.GetClassification(itemId) == item.ClassificationConsumablePetFood {
		_ = consumable.ConsumePetFood(l)(ctx)(e.CharacterId, e.Body.ItemId, e.Slot, e.Body.Quantity, e.Body.TransactionId)
		return
	}

	_ = inventory2.CancelItemReservation(l)(ctx)(e.CharacterId, inventory.TypeValueUse, e.Body.TransactionId, e.Slot)
}
