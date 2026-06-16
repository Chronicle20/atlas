package summon

import (
	consumer2 "atlas-summons/kafka/consumer"
	"atlas-summons/summon"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("summon_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAttackCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleSpawnCommand(l logrus.FieldLogger, ctx context.Context, c Command[SpawnCommandBody]) {
	if c.Type != CommandTypeSpawn {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, err := summon.NewProcessor(l, ctx).Spawn(f, c.Body.OwnerCharacterId, c.Body.SkillId, c.Body.SkillLevel, c.Body.X, c.Body.Y, c.Body.AuraLevel, c.Body.HexLevel)
	if err != nil {
		l.WithError(err).Errorf("Failed to spawn summon for owner [%d] skill [%d].", c.Body.OwnerCharacterId, c.Body.SkillId)
	}
}

func handleMoveCommand(l logrus.FieldLogger, ctx context.Context, c Command[MoveCommandBody]) {
	if c.Type != CommandTypeMove {
		return
	}
	err := summon.NewProcessor(l, ctx).Move(c.Body.SummonId, c.Body.SenderCharacterId, c.Body.X, c.Body.Y, c.Body.Stance, c.Body.RawMovement)
	if err != nil {
		l.WithError(err).Errorf("Failed to move summon [%d] for sender [%d].", c.Body.SummonId, c.Body.SenderCharacterId)
	}
}

func handleAttackCommand(l logrus.FieldLogger, ctx context.Context, c Command[AttackCommandBody]) {
	if c.Type != CommandTypeAttack {
		return
	}
	targets := make([]summon.AttackTarget, 0, len(c.Body.Targets))
	for _, t := range c.Body.Targets {
		targets = append(targets, summon.AttackTarget{MonsterId: t.MonsterId, Damage: t.Damage})
	}
	err := summon.NewProcessor(l, ctx).Attack(c.Body.SummonId, c.Body.SenderCharacterId, c.Body.Direction, targets)
	if err != nil {
		l.WithError(err).Errorf("Failed to attack with summon [%d] for sender [%d].", c.Body.SummonId, c.Body.SenderCharacterId)
	}
}

func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c Command[DamageCommandBody]) {
	if c.Type != CommandTypeDamage {
		return
	}
	err := summon.NewProcessor(l, ctx).Damage(c.Body.SummonId, c.Body.SenderCharacterId, c.Body.Damage, c.Body.MonsterIdFrom)
	if err != nil {
		l.WithError(err).Errorf("Failed to damage summon [%d] for sender [%d].", c.Body.SummonId, c.Body.SenderCharacterId)
	}
}
