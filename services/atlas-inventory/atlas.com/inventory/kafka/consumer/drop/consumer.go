package drop

import (
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	"atlas-inventory/kafka/message/drop"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
		m := _map.NewModel(world.Id(e.WorldId))(channel.Id(e.ChannelId))(_map.Id(e.MapId))
		if e.Body.EquipmentId > 0 {
			// TODO this needs to be added to drop event
			_ = compartment.NewProcessor(l, ctx, db).AttemptEquipmentPickUpAndEmit(uuid.New(), m, e.Body.CharacterId, e.DropId, e.Body.ItemId, e.Body.EquipmentId)
			return
		}
		if e.Body.ItemId > 0 {
			// TODO this needs to be added to drop event
			_ = compartment.NewProcessor(l, ctx, db).AttemptItemPickUpAndEmit(uuid.New(), m, e.Body.CharacterId, e.DropId, e.Body.ItemId, e.Body.Quantity)
			return
		}
	}
}
