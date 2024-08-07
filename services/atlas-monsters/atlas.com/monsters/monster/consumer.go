package monster

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/kafka/producer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func DamageConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameDamage)(EnvCommandTopicDamage)(groupId)
	}
}

func DamageCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicDamage)()
	return t, message.AdaptHandler(message.PersistentConfig(handleDamageCommand))
}

func handleDamageCommand(l logrus.FieldLogger, span opentracing.Span, command damageCommand) {
	Damage(l, span, command.Tenant)(command.UniqueId, command.CharacterId, command.Damage)
}

func MovementConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameMovement)(EnvCommandTopicMovement)(groupId)
	}
}

func MovementCommandRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicMovement)()
	return t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand))
}

func handleMovementCommand(l logrus.FieldLogger, span opentracing.Span, command movementCommand) {
	endX := int16(0)
	endY := int16(0)
	stance := byte(0)
	for _, m := range command.Movement.Elements {
		if m.TypeStr == MovementTypeNormal {
			endX = m.X
			endY = m.Y
			stance = m.MoveAction
		}
	}

	Move(l, span, command.Tenant)(command.UniqueId, endX, endY, stance)

	err := producer.ProviderImpl(l)(span)(EnvEventTopicMovement)(emitMove(command.Tenant, command.WorldId, command.ChannelId, command.UniqueId, command.ObserverId, command.SkillPossible, command.Skill, command.SkillId, command.SkillLevel, command.MultiTarget, command.RandomTimes, command.Movement))
	if err != nil {
		l.WithError(err).Errorf("Unable to relay monster [%d] movement to other characters in map.", command.UniqueId)
	}
}
