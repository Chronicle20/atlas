package skill

import (
	consumer2 "atlas-skills/kafka/consumer"
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/skill"
	"context"
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
			rf(consumer2.NewConfig(l)("skill_command")(skill2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(skill2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestUpdate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandSetCooldown(db))))
		}
	}
}

func handleCommandRequestCreate(db *gorm.DB) message.Handler[skill2.Command[skill2.RequestCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c skill2.Command[skill2.RequestCreateBody]) {
		if c.Type != skill2.CommandTypeRequestCreate {
			return
		}

		_, err := skill.NewProcessor(l, ctx, db).CreateAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.SkillId, c.Body.Level, c.Body.MasterLevel, c.Body.Expiration)
		if err != nil {
			l.WithError(err).Errorf("Unable to create skill [%d] for character [%d].", c.Body.SkillId, c.CharacterId)
		}
	}
}

func handleCommandRequestUpdate(db *gorm.DB) message.Handler[skill2.Command[skill2.RequestUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c skill2.Command[skill2.RequestUpdateBody]) {
		if c.Type != skill2.CommandTypeRequestUpdate {
			return
		}

		_, err := skill.NewProcessor(l, ctx, db).UpdateAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.SkillId, c.Body.Level, c.Body.MasterLevel, c.Body.Expiration)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill [%d] for character [%d].", c.Body.SkillId, c.CharacterId)
		}
	}
}

func handleCommandSetCooldown(db *gorm.DB) message.Handler[skill2.Command[skill2.SetCooldownBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c skill2.Command[skill2.SetCooldownBody]) {
		if c.Type != skill2.CommandTypeSetCooldown {
			return
		}

		_, _ = skill.NewProcessor(l, ctx, db).SetCooldownAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.SkillId, c.Body.Cooldown)
	}
}
