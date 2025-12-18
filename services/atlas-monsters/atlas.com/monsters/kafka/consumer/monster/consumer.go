package monster

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/monster"
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
			rf(consumer2.NewConfig(l)("monster_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("monster_movement_event")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageCommand)))
		t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand)))
	}
}

func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c command[damageCommandBody]) {
	if c.Type != CommandTypeDamage {
		return
	}

	monster.Damage(l)(ctx)(c.MonsterId, c.Body.CharacterId, c.Body.Damage)
}

func handleMovementCommand(_ logrus.FieldLogger, ctx context.Context, c movementCommand) {
	_ = monster.Move(ctx)(uint32(c.ObjectId), c.X, c.Y, c.Stance)
}
