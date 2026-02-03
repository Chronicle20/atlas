package expression

import (
	"atlas-expressions/expression"
	consumer2 "atlas-expressions/kafka/consumer"
	expressionMsg "atlas-expressions/kafka/message/expression"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("expression_command")(expressionMsg.EnvExpressionCommand)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(expressionMsg.EnvExpressionCommand)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeCommand)))
	}
}

func handleChangeCommand(l logrus.FieldLogger, ctx context.Context, c expressionMsg.Command) {
	processor := expression.NewProcessor(l, ctx)
	_, _ = processor.ChangeAndEmit(c.TransactionId, c.CharacterId, c.WorldId, c.ChannelId, c.MapId, c.Instance, c.Expression)
}
