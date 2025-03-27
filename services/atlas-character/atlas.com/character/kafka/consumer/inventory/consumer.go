package inventory

import (
	"atlas-character/character"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/inventory"
	consumer2 "atlas-character/kafka/consumer"
	"atlas-character/kafka/producer"
	"context"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
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
			rf(consumer2.NewConfig(l)("inventory_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDropItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestReserveItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleConsumeItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyItemCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelItemReservationCommand(db))))
		}
	}
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[command[equipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[equipCommandBody]) {
		if c.Type != CommandEquip {
			return
		}

		l.Debugf("Received equip item command. characterId [%d] source [%d] destination [%d]", c.CharacterId, c.Body.Source, c.Body.Destination)
		fsp := model.Flip(equipable.GetNextFreeSlot(l))(ctx)
		ep := producer.ProviderImpl(l)(ctx)
		dp := equipment.GetEquipmentDestination(l)(ctx)(c.Body.Destination)
		inventory.EquipItemForCharacter(l)(db)(ctx)(fsp)(ep)(c.CharacterId)(c.Body.Source)(dp)
	}
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[command[unequipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[unequipCommandBody]) {
		if c.Type != CommandUnequip {
			return
		}

		l.Debugf("Received unequip item command. characterId [%d] source [%d].", c.CharacterId, c.Body.Source)
		fsp := model.Flip(equipable.GetNextFreeSlot(l))(ctx)
		ep := producer.ProviderImpl(l)(ctx)
		inventory.UnequipItemForCharacter(l)(db)(ctx)(fsp)(ep)(c.CharacterId)(c.Body.Source)
	}
}

func handleMoveItemCommand(db *gorm.DB) message.Handler[command[moveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[moveCommandBody]) {
		if c.Type != CommandMove {
			return
		}

		_ = inventory.Move(l)(db)(ctx)(producer.ProviderImpl(l)(ctx))(inventory2.Type(c.InventoryType))(c.CharacterId)(c.Body.Source)(c.Body.Destination)
	}
}

func handleDropItemCommand(db *gorm.DB) message.Handler[command[dropCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[dropCommandBody]) {
		if c.Type != CommandDrop {
			return
		}

		td := character.GetTemporalRegistry().GetById(c.CharacterId)
		_ = inventory.Drop(l)(db)(ctx)(inventory2.Type(c.InventoryType))(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.CharacterId, td.X(), td.Y(), c.Body.Source, c.Body.Quantity)
	}
}

func handleRequestReserveItemCommand(db *gorm.DB) message.Handler[command[requestReserveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestReserveCommandBody]) {
		if c.Type != CommandRequestReserve {
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

func handleConsumeItemCommand(db *gorm.DB) message.Handler[command[consumeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[consumeCommandBody]) {
		if c.Type != CommandConsume {
			return
		}
		_ = inventory.ConsumeItem(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.TransactionId, c.Body.Slot)
	}
}

func handleDestroyItemCommand(db *gorm.DB) message.Handler[command[destroyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[destroyCommandBody]) {
		if c.Type != CommandDestroy {
			return
		}
		_ = inventory.DestroyItem(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.Slot)
	}
}

func handleCancelItemReservationCommand(db *gorm.DB) message.Handler[command[cancelReservationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[cancelReservationCommandBody]) {
		if c.Type != CommandCancelReservation {
			return
		}
		_ = inventory.CancelReservation(l)(ctx)(db)(c.CharacterId, inventory2.Type(c.InventoryType), c.Body.TransactionId, c.Body.Slot)
	}
}
