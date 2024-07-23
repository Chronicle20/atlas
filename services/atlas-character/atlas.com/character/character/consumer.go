package character

import (
	"atlas-character/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerCommand = "character_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func ChangeMapCommandRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvCommandTopic), message.AdaptHandler(message.PersistentConfig(handleChangeMap(db)))
}

func handleChangeMap(db *gorm.DB) func(l logrus.FieldLogger, span opentracing.Span, command commandEvent[changeMapBody]) {
	return func(l logrus.FieldLogger, span opentracing.Span, command commandEvent[changeMapBody]) {
		ChangeMap(l, db, span, command.Tenant)(command.CharacterId, command.WorldId, command.Body.ChannelId, command.Body.MapId, command.Body.PortalId)
	}
}
