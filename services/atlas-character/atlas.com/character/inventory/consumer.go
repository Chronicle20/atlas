package inventory

import (
	"atlas-character/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerEquipItem   = "equip_item_command"
	consumerUnequipItem = "unequip_item_command"
)

func EquipItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerEquipItem)(EnvCommandTopicEquipItem)(groupId)
	}
}

func EquipItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvCommandTopicEquipItem), message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db)))
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[equipItemCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command equipItemCommand) {
		l.Debugf("Received equip item command. characterId [%d] source [%d] destination [%d]", command.CharacterId, command.Source, command.Destination)
		EquipItemForCharacter(l, db, span, command.Tenant)(command.CharacterId, command.Source, command.Destination)
	}
}

func UnequipItemCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerUnequipItem)(EnvCommandTopicUnequipItem)(groupId)
	}
}

func UnequipItemRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvCommandTopicUnequipItem), message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db)))
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[unequipItemCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command unequipItemCommand) {
		l.Debugf("Received unequip item command. characterId [%d] source [%d].", command.CharacterId, command.Source)
		UnequipItemForCharacter(l, db, span, command.Tenant)(command.CharacterId, command.Source)
	}
}
