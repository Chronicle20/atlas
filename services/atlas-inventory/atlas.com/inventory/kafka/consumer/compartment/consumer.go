package compartment

import (
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	compartment2 "atlas-inventory/kafka/message/compartment"
	"atlas-inventory/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("compartment_command")(compartment2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(compartment2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDropItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestReserveItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleConsumeItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelItemReservationCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleIncreaseCapacityCommand(db))))
		}
	}
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.EquipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.EquipCommandBody]) {
		if c.Type != compartment2.CommandEquip {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).EquipItemAndEmit(c.CharacterId, c.Body.Source, c.Body.Destination)
	}
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.UnequipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.UnequipCommandBody]) {
		if c.Type != compartment2.CommandUnequip {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).RemoveEquipAndEmit(c.CharacterId, c.Body.Source, c.Body.Destination)
	}
}

func handleMoveItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.MoveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.MoveCommandBody]) {
		if c.Type != compartment2.CommandMove {
			return
		}

		_ = inventory.Move(l)(db)(ctx)(producer.ProviderImpl(l)(ctx))(inventory2.Type(c.InventoryType))(c.CharacterId)(c.Body.Source)(c.Body.Destination)
	}
}

func handleDropItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.DropCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.DropCommandBody]) {
		if c.Type != compartment2.CommandDrop {
			return
		}

		td := character.GetTemporalRegistry().GetById(c.CharacterId)
		_ = inventory.Drop(l)(db)(ctx)(inventory2.Type(c.InventoryType))(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.CharacterId, td.X(), td.Y(), c.Body.Source, c.Body.Quantity)
	}
}

func handleRequestReserveItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.RequestReserveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.RequestReserveCommandBody]) {
		if c.Type != compartment2.CommandRequestReserve {
			return
		}
		reserves := make([]inventory.Reserve, 0)
		for _, i := range c.Body.Items {
			reserves = append(reserves, inventory.Reserve{
				Slot:     i.Source,
				ItemId:   i.ItemId,
				Quantity: i.Quantity,
			})
		}

		_ = inventory.RequestReserve(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), reserves, c.Body.TransactionId)
	}
}

func handleConsumeItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ConsumeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ConsumeCommandBody]) {
		if c.Type != compartment2.CommandConsume {
			return
		}
		_ = inventory.ConsumeItem(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.TransactionId, c.Body.Slot)
	}
}

func handleDestroyItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.DestroyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.DestroyCommandBody]) {
		if c.Type != compartment2.CommandDestroy {
			return
		}
		_ = inventory.DestroyItem(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.Slot)
	}
}

func handleCancelItemReservationCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.CancelReservationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.CancelReservationCommandBody]) {
		if c.Type != compartment2.CommandCancelReservation {
			return
		}
		_ = inventory.CancelReservation(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.TransactionId, c.Body.Slot)
	}
}

func handleIncreaseCapacityCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.IncreaseCapacityCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.IncreaseCapacityCommandBody]) {
		if c.Type != compartment2.CommandIncreaseCapacity {
			return
		}
		_ = inventory.IncreaseCapacity(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.Amount)
	}
}
