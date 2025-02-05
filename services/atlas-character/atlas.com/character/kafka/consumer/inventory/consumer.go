package inventory

import (
	"atlas-character/character"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/inventory"
	consumer2 "atlas-character/kafka/consumer"
	"atlas-character/kafka/producer"
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
			rf(consumer2.NewConfig(l)("equip_item_command")(EnvCommandTopicEquipItem)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("unequip_item_command")(EnvCommandTopicUnequipItem)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("move_item_command")(EnvCommandTopicMoveItem)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("drop_item_command")(EnvCommandTopicDropItem)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopicEquipItem)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicUnequipItem)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicMoveItem)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveItemCommand(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicDropItem)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDropItemCommand(db))))
		}
	}
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[equipItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c equipItemCommand) {
		l.Debugf("Received equip item command. characterId [%d] source [%d] destination [%d]", c.CharacterId, c.Source, c.Destination)
		fsp := model.Flip(equipable.GetNextFreeSlot(l))(ctx)
		ep := producer.ProviderImpl(l)(ctx)
		dp := equipment.GetEquipmentDestination(l)(ctx)
		inventory.EquipItemForCharacter(l)(db)(ctx)(fsp)(ep)(c.CharacterId)(c.Source)(dp)
	}
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[unequipItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c unequipItemCommand) {
		l.Debugf("Received unequip item command. characterId [%d] source [%d].", c.CharacterId, c.Source)
		fsp := model.Flip(equipable.GetNextFreeSlot(l))(ctx)
		ep := producer.ProviderImpl(l)(ctx)
		inventory.UnequipItemForCharacter(l)(db)(ctx)(fsp)(ep)(c.CharacterId)(c.Source)
	}
}

func handleMoveItemCommand(db *gorm.DB) message.Handler[moveItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c moveItemCommand) {
		_ = inventory.Move(l)(db)(ctx)(producer.ProviderImpl(l)(ctx))(c.InventoryType)(c.CharacterId)(c.Source)(c.Destination)
	}
}

func handleDropItemCommand(db *gorm.DB) message.Handler[dropItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c dropItemCommand) {
		td := character.GetTemporalRegistry().GetById(c.CharacterId)
		_ = inventory.Drop(l)(db)(ctx)(c.InventoryType)(c.WorldId, c.ChannelId, c.MapId, c.CharacterId, td.X(), td.Y(), c.Source, c.Quantity)
	}
}
