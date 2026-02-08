package drop

import (
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	"atlas-inventory/kafka/message/drop"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_status_event")(drop.EnvEventTopicDropStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(drop.EnvEventTopicDropStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDropReservation(db))))

		}
	}
}

func handleDropReservation(db *gorm.DB) message.Handler[drop.StatusEvent[drop.ReservedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e drop.StatusEvent[drop.ReservedStatusEventBody]) {
		if e.Type != drop.StatusEventTypeReserved {
			return
		}
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		if e.Body.ItemId > 0 {
			if it, ok := inventory.TypeFromItemId(item.Id(e.Body.ItemId)); ok && it == inventory.TypeValueEquip {
				_ = compartment.NewProcessor(l, ctx, db).AttemptEquipmentPickUpAndEmit(uuid.New(), f, e.Body.CharacterId, e.DropId, e.Body.ItemId, e.Body.EquipmentData)
			} else {
				_ = compartment.NewProcessor(l, ctx, db).AttemptItemPickUpAndEmit(uuid.New(), f, e.Body.CharacterId, e.DropId, e.Body.ItemId, e.Body.Quantity)
			}
		}
	}
}
