package skill

import (
	consumer2 "atlas-skills/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("skill_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestUpdate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandSetCooldown(db))))
		}
	}
}

func handleCommandRequestCreate(db *gorm.DB) message.Handler[command[requestCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestCreateBody]) {
		if c.Type != CommandTypeRequestCreate {
			return
		}

		_, err := skill.Create(l)(ctx)(db)(c.CharacterId, c.Body.SkillId, c.Body.Level, c.Body.MasterLevel, c.Body.Expiration)
		if err != nil {
			l.WithError(err).Errorf("Unable to create skill [%d] for character [%d].", c.Body.SkillId, c.CharacterId)
		}
	}
}

func handleCommandRequestUpdate(db *gorm.DB) message.Handler[command[requestUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestUpdateBody]) {
		if c.Type != CommandTypeRequestUpdate {
			return
		}

		_, err := skill.Update(l)(ctx)(db)(c.CharacterId, c.Body.SkillId, c.Body.Level, c.Body.MasterLevel, c.Body.Expiration)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill [%d] for character [%d].", c.Body.SkillId, c.CharacterId)
		}
	}
}

func handleCommandSetCooldown(db *gorm.DB) message.Handler[command[setCooldownBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[setCooldownBody]) {
		if c.Type != CommandTypeSetCooldown {
			return
		}

		_, _ = skill.SetCooldown(l)(ctx)(db)(c.CharacterId, c.Body.SkillId, c.Body.Cooldown)
	}
}
