package portal

import (
	"atlas-portals/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerCommand = "portal_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerCommand)(EnvPortalCommandTopic)(groupId)
	}
}

func EnterCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvPortalCommandTopic), message.AdaptHandler(message.PersistentConfig(handleEnterCommand))
}

func handleEnterCommand(l logrus.FieldLogger, span opentracing.Span, command commandEvent[enterBody]) {
	l.Debugf("Received command for Character [%d] to enter portal [%d] in map [%d].", command.Body.CharacterId, command.PortalId, command.MapId)
	Enter(l, span, command.Tenant)(command.WorldId, command.ChannelId, command.MapId, command.PortalId, command.Body.CharacterId)
}
