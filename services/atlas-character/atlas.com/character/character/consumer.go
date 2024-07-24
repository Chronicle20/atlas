package character

import (
	consumer2 "atlas-character/kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerCommand = "character_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func ChangeMapCommandRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleChangeMap(db)))
}

func handleChangeMap(db *gorm.DB) func(l logrus.FieldLogger, span opentracing.Span, command commandEvent[changeMapBody]) {
	return func(l logrus.FieldLogger, span opentracing.Span, command commandEvent[changeMapBody]) {
		ChangeMap(l, db, span, command.Tenant)(command.CharacterId, command.WorldId, command.Body.ChannelId, command.Body.MapId, command.Body.PortalId)
	}
}
