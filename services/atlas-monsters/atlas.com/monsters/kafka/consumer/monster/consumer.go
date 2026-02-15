package monster

import (
	consumer2 "atlas-monsters/kafka/consumer"
	"atlas-monsters/monster"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
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
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleApplyStatusCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelStatusCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUseSkillCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleApplyStatusFieldCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelStatusFieldCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUseSkillFieldCommand)))
		t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand)))
	}
}

func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c command[damageCommandBody]) {
	if c.Type != CommandTypeDamage {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.Damage(c.MonsterId, c.Body.CharacterId, c.Body.Damage, c.Body.AttackType)
}

func handleApplyStatusCommand(l logrus.FieldLogger, ctx context.Context, c command[applyStatusCommandBody]) {
	if c.Type != CommandTypeApplyStatus {
		return
	}

	tickInterval := time.Duration(c.Body.TickInterval) * time.Millisecond
	// Auto-set tick interval for DoT statuses
	if tickInterval == 0 {
		for statusType := range c.Body.Statuses {
			if statusType == "POISON" || statusType == "VENOM" {
				tickInterval = 1000 * time.Millisecond
				break
			}
		}
	}

	effect := monster.NewStatusEffect(
		c.Body.SourceType,
		c.Body.SourceCharacterId,
		c.Body.SourceSkillId,
		c.Body.SourceSkillLevel,
		c.Body.Statuses,
		time.Duration(c.Body.Duration)*time.Millisecond,
		tickInterval,
	)

	p := monster.NewProcessor(l, ctx)
	_ = p.ApplyStatusEffect(c.MonsterId, effect)
}

func handleCancelStatusCommand(l logrus.FieldLogger, ctx context.Context, c command[cancelStatusCommandBody]) {
	if c.Type != CommandTypeCancelStatus {
		return
	}

	p := monster.NewProcessor(l, ctx)
	if len(c.Body.StatusTypes) == 0 {
		_ = p.CancelAllStatusEffects(c.MonsterId)
	} else {
		_ = p.CancelStatusEffect(c.MonsterId, c.Body.StatusTypes)
	}
}

func handleUseSkillCommand(l logrus.FieldLogger, ctx context.Context, c command[useSkillCommandBody]) {
	if c.Type != CommandTypeUseSkill {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.UseSkill(c.MonsterId, c.Body.CharacterId, c.Body.SkillId, c.Body.SkillLevel)
}

func handleMovementCommand(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
	p := monster.NewProcessor(l, ctx)
	_ = p.Move(uint32(c.ObjectId), c.X, c.Y, c.Stance)
}

func handleApplyStatusFieldCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[applyStatusCommandBody]) {
	if c.Type != CommandTypeApplyStatusField {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	p := monster.NewProcessor(l, ctx)
	monsters, err := p.GetInField(f)
	if err != nil {
		l.WithError(err).Errorf("Unable to get monsters in field for field-level status apply.")
		return
	}

	tickInterval := time.Duration(c.Body.TickInterval) * time.Millisecond
	if tickInterval == 0 {
		for statusType := range c.Body.Statuses {
			if statusType == "POISON" || statusType == "VENOM" {
				tickInterval = 1000 * time.Millisecond
				break
			}
		}
	}

	for _, m := range monsters {
		effect := monster.NewStatusEffect(
			c.Body.SourceType,
			c.Body.SourceCharacterId,
			c.Body.SourceSkillId,
			c.Body.SourceSkillLevel,
			c.Body.Statuses,
			time.Duration(c.Body.Duration)*time.Millisecond,
			tickInterval,
		)
		_ = p.ApplyStatusEffect(m.UniqueId(), effect)
	}
}

func handleCancelStatusFieldCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[cancelStatusCommandBody]) {
	if c.Type != CommandTypeCancelStatusField {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	p := monster.NewProcessor(l, ctx)
	monsters, err := p.GetInField(f)
	if err != nil {
		l.WithError(err).Errorf("Unable to get monsters in field for field-level status cancel.")
		return
	}

	for _, m := range monsters {
		if len(c.Body.StatusTypes) == 0 {
			_ = p.CancelAllStatusEffects(m.UniqueId())
		} else {
			_ = p.CancelStatusEffect(m.UniqueId(), c.Body.StatusTypes)
		}
	}
}

func handleUseSkillFieldCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[useSkillFieldCommandBody]) {
	if c.Type != CommandTypeUseSkillField {
		return
	}

	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	p := monster.NewProcessor(l, ctx)
	monsters, err := p.GetInField(f)
	if err != nil {
		l.WithError(err).Errorf("Unable to get monsters in field for field-level skill use.")
		return
	}

	for _, m := range monsters {
		p.UseSkillGM(m.UniqueId(), c.Body.SkillId, c.Body.SkillLevel)
	}
}
