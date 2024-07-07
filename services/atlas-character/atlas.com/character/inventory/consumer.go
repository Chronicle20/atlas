package inventory

import (
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerEquipItem   = "equip_item_command"
	consumerUnequipItem = "unequip_item_command"
)

func EquipItemCommandConsumer(l logrus.FieldLogger, db *gorm.DB) func(groupId string) consumer.Config {
	t := lookupTopic(l)(EnvCommandTopicEquipItem)
	return func(groupId string) consumer.Config {
		return consumer.NewConfig[equipItemCommand](consumerEquipItem, t, groupId, handleEquipItemCommand(db))
	}
}

func handleEquipItemCommand(db *gorm.DB) consumer.HandlerFunc[equipItemCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command equipItemCommand) {
		l.Debugf("Received equip item command. characterId [%d] source [%d] destination [%d]", command.CharacterId, command.Source, command.Destination)
		EquipItemForCharacter(l, db, span, command.Tenant)(command.CharacterId, command.Source, command.Destination)
	}
}

func UnequipItemCommandConsumer(l logrus.FieldLogger, db *gorm.DB) func(groupId string) consumer.Config {
	t := lookupTopic(l)(EnvCommandTopicUnequipItem)
	return func(groupId string) consumer.Config {
		return consumer.NewConfig[unequipItemCommand](consumerUnequipItem, t, groupId, handleUnequipItemCommand(db))
	}
}

func handleUnequipItemCommand(db *gorm.DB) consumer.HandlerFunc[unequipItemCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command unequipItemCommand) {
		l.Debugf("Received unequip item command. characterId [%d] source [%d].", command.CharacterId, command.Source)
		UnequipItemForCharacter(l, db, span, command.Tenant)(command.CharacterId, command.Source)
	}
}
