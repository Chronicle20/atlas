package inventory

import (
	"atlas-character/equipable"
	"atlas-character/equipment"
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

const (
	consumerEquipItem   = "equip_item_command"
	consumerUnequipItem = "unequip_item_command"
	consumerMoveItem    = "move_item_command"
	consumerDropItem    = "drop_item_command"
)

func EquipItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerEquipItem)(EnvCommandTopicEquipItem)(groupId)
	}
}

func EquipItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicEquipItem)()
	return t, message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db)))
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[equipItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, command equipItemCommand) {
		l.Debugf("Received equip item command. characterId [%d] source [%d] destination [%d]", command.CharacterId, command.Source, command.Destination)
		fsp := model.Flip(model.Flip(equipable.GetNextFreeSlot(l))(ctx))(command.Tenant)
		ep := producer.ProviderImpl(l)(ctx)
		dp := equipment.GetEquipmentDestination(l)(ctx)(command.Tenant)
		EquipItemForCharacter(l)(db)(command.Tenant)(fsp)(ep)(command.CharacterId)(command.Source)(dp)
	}
}

func UnequipItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerUnequipItem)(EnvCommandTopicUnequipItem)(groupId)
	}
}

func UnequipItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicUnequipItem)()
	return t, message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db)))
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[unequipItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, command unequipItemCommand) {
		l.Debugf("Received unequip item command. characterId [%d] source [%d].", command.CharacterId, command.Source)
		fsp := model.Flip(model.Flip(equipable.GetNextFreeSlot(l))(ctx))(command.Tenant)
		ep := producer.ProviderImpl(l)(ctx)
		UnequipItemForCharacter(l)(db)(command.Tenant)(fsp)(ep)(command.CharacterId)(command.Source)
	}
}

func MoveItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerMoveItem)(EnvCommandTopicMoveItem)(groupId)
	}
}

func MoveItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicMoveItem)()
	return t, message.AdaptHandler(message.PersistentConfig(handleMoveItemCommand(db)))
}

func handleMoveItemCommand(db *gorm.DB) message.Handler[moveItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, command moveItemCommand) {
		_ = Move(l, db, producer.ProviderImpl(l)(ctx))(command.Tenant, command.CharacterId, command.InventoryType, command.Source, command.Destination)
	}
}

func DropItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerDropItem)(EnvCommandTopicDropItem)(groupId)
	}
}

func DropItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicDropItem)()
	return t, message.AdaptHandler(message.PersistentConfig(handleDropItemCommand(db)))
}

func handleDropItemCommand(db *gorm.DB) message.Handler[dropItemCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, command dropItemCommand) {
		_ = Drop(l, db, ctx, command.Tenant)(command.CharacterId, command.InventoryType, command.Source, command.Quantity)
	}
}
