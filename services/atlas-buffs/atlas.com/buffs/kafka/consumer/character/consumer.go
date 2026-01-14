package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/character"
	consumer2 "atlas-buffs/kafka/consumer"
	character2 "atlas-buffs/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("buff_command")(character2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleApply)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancel)))
	}
}

func handleApply(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ApplyCommandBody]) {
	if c.Type != character2.CommandTypeApply {
		return
	}

	statChanges := make([]stat.Model, 0)
	for _, cs := range c.Body.Changes {
		statChanges = append(statChanges, stat.NewStat(cs.Type, cs.Amount))
	}

	if err := character.NewProcessor(l, ctx).Apply(c.WorldId, c.CharacterId, c.Body.FromId, c.Body.SourceId, c.Body.Duration, statChanges); err != nil {
		l.WithError(err).Errorf("Unable to apply buff [%d] to character [%d].", c.Body.SourceId, c.CharacterId)
	}
}

func handleCancel(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CancelCommandBody]) {
	if c.Type != character2.CommandTypeCancel {
		return
	}

	if err := character.NewProcessor(l, ctx).Cancel(c.WorldId, c.CharacterId, c.Body.SourceId); err != nil {
		l.WithError(err).Errorf("Unable to cancel buff [%d] for character [%d].", c.Body.SourceId, c.CharacterId)
	}
}
