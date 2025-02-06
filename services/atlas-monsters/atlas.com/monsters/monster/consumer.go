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

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_damage_event")(EnvCommandTopicDamage)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("monster_movement_event")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopicDamage)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageCommand)))
		t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand)))
	}
}

func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, command damageCommand) {
	Damage(l)(ctx)(command.UniqueId, command.CharacterId, command.Damage)
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

	Move(ctx)(command.UniqueId, ms.X, ms.Y, ms.Stance)

	err = producer.ProviderImpl(l)(ctx)(EnvEventTopicMovement)(emitMove(command.WorldId, command.ChannelId, command.UniqueId, command.ObserverId, command.SkillPossible, command.Skill, command.SkillId, command.SkillLevel, command.MultiTarget, command.RandomTimes, command.Movement))
	if err != nil {
		l.WithError(err).Errorf("Unable to relay monster [%d] movement to other characters in map.", command.UniqueId)
	}
}
