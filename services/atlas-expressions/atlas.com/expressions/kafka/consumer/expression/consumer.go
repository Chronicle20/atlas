package expression

import (
	"atlas-expressions/expression"
	consumer2 "atlas-expressions/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)("expression_command")(EnvExpressionCommand)(groupId)
	}
}

func ChangeCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvExpressionCommand)()
	return t, message.AdaptHandler(message.PersistentConfig(handleChangeCommand))
}

func handleChangeCommand(l logrus.FieldLogger, ctx context.Context, c expressionCommand) {
	_ = expression.Change(l)(ctx)(c.CharacterId, c.WorldId, c.ChannelId, c.MapId, c.Expression)
}
