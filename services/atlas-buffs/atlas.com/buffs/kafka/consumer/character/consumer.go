package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/character"
	consumer2 "atlas-buffs/kafka/consumer"
	character2 "atlas-buffs/kafka/message/character"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("buff_command")(character2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleApply))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancel))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelAll))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelByTypes))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateStatValue))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExpire))); err != nil {
			return err
		}
		return nil
	}
}

func handleApply(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ApplyCommandBody]) {
	if c.Type != character2.CommandTypeApply {
		return
	}

	if c.Body.NoExpiry && c.Body.Duration != 0 {
		l.Warnf("Rejecting malformed APPLY for character [%d] source [%d]: noExpiry with nonzero duration [%d].", c.CharacterId, c.Body.SourceId, c.Body.Duration)
		return
	}

	statChanges := make([]stat.Model, 0)
	for _, cs := range c.Body.Changes {
		statChanges = append(statChanges, stat.NewStat(cs.Type, cs.Amount))
	}

	if err := character.NewProcessor(l, ctx).Apply(c.WorldId, c.ChannelId, c.CharacterId, c.Body.FromId, c.Body.SourceId, c.Body.Level, c.Body.Duration, statChanges, c.Body.Accumulate, c.Body.NoExpiry); err != nil {
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

func handleCancelAll(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CancelAllCommandBody]) {
	if c.Type != character2.CommandTypeCancelAll {
		return
	}

	if err := character.NewProcessor(l, ctx).CancelAll(c.WorldId, c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to cancel all buffs for character [%d].", c.CharacterId)
	}
}

func handleCancelByTypes(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CancelByTypesCommandBody]) {
	if c.Type != character2.CommandTypeCancelByTypes {
		return
	}

	if err := character.NewProcessor(l, ctx).CancelByStatTypes(c.WorldId, c.CharacterId, c.Body.Types); err != nil {
		l.WithError(err).Errorf("Unable to cancel buffs by types %v for character [%d].", c.Body.Types, c.CharacterId)
	}
}

func handleUpdateStatValue(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.UpdateStatValueCommandBody]) {
	if c.Type != character2.CommandTypeUpdateStatValue {
		return
	}

	u := character.StatValueUpdate{
		SourceId:        c.Body.SourceId,
		StatType:        c.Body.StatType,
		Operation:       c.Body.Operation,
		Amount:          c.Body.Amount,
		Cap:             c.Body.Cap,
		CreateIfMissing: c.Body.CreateIfMissing,
		Level:           c.Body.Level,
	}
	if err := character.NewProcessor(l, ctx).UpdateStatValue(c.WorldId, c.ChannelId, c.CharacterId, u); err != nil {
		l.WithError(err).Errorf("Unable to update stat value on buff [%d] for character [%d].", c.Body.SourceId, c.CharacterId)
	}
}

// handleExpire answers a character's CANCEL_DEBUFF nudge with a per-character
// expiry sweep. Nothing lapsed ⇒ nothing emitted (task-190 FR-2.9).
func handleExpire(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ExpireCommandBody]) {
	if c.Type != character2.CommandTypeExpire {
		return
	}

	if err := character.NewProcessor(l, ctx).ExpireForCharacter(c.WorldId, c.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to expire buffs for character [%d].", c.CharacterId)
	}
}
