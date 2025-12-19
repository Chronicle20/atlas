package macro

import (
	consumer2 "atlas-skills/kafka/consumer"
	macro2 "atlas-skills/kafka/message/macro"
	"atlas-skills/macro"
	"context"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
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
			rf(consumer2.NewConfig(l)("skill_macro_command")(macro2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(macro2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandUpdate(db))))
		}
	}
}

func handleCommandUpdate(db *gorm.DB) message.Handler[macro2.Command[macro2.UpdateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c macro2.Command[macro2.UpdateCommandBody]) {
		if c.Type != macro2.CommandTypeUpdate {
			return
		}

		macros := make([]macro.Model, 0)
		for _, m := range c.Body.Macros {
			macros = append(macros, macro.NewModel(m.Id, m.Name, m.Shout, skill2.Id(m.SkillId1), skill2.Id(m.SkillId2), skill2.Id(m.SkillId3)))
		}

		processor := macro.NewProcessor(l, ctx, db)
		_, err := processor.UpdateAndEmit(c.CharacterId, macros)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", c.CharacterId)
		}
	}
}
