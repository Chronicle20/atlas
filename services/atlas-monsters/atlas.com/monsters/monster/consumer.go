package monster

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
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

func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, command damageCommand) {
	Damage(l, ctx, command.Tenant)(command.UniqueId, command.CharacterId, command.Damage)
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

type MovementSummary struct {
	X      int16
	Y      int16
	Stance byte
}

func MovementSummaryProvider() (MovementSummary, error) {
	return MovementSummary{}, nil
}

func FoldMovement(summary MovementSummary, e element) (MovementSummary, error) {
	res := MovementSummary{
		X:      summary.X,
		Y:      summary.Y,
		Stance: summary.Stance,
	}
	if e.TypeStr == MovementTypeNormal {
		res.X = e.X
		res.Y = e.Y
		res.Stance = e.MoveAction
	}
	return res, nil
}

func handleMovementCommand(l logrus.FieldLogger, ctx context.Context, command movementCommand) {
	ms, err := model.Fold(model.FixedProvider(command.Movement.Elements), MovementSummaryProvider, FoldMovement)()
	if err != nil {
		return
	}

	Move(command.Tenant)(command.UniqueId, ms.X, ms.Y, ms.Stance)

	err = producer.ProviderImpl(l)(ctx)(EnvEventTopicMovement)(emitMove(command.Tenant, command.WorldId, command.ChannelId, command.UniqueId, command.ObserverId, command.SkillPossible, command.Skill, command.SkillId, command.SkillLevel, command.MultiTarget, command.RandomTimes, command.Movement))
	if err != nil {
		l.WithError(err).Errorf("Unable to relay monster [%d] movement to other characters in map.", command.UniqueId)
	}
}
