package consumable

import (
	"atlas-consumables/consumable"
	consumer2 "atlas-consumables/kafka/consumer"
	consumable2 "atlas-consumables/kafka/message/consumable"
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
			rf(consumer2.NewConfig(l)("consumable_command")(consumable2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(consumable2.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestItemConsume)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestScroll)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleApplyConsumableEffect)))
	}
}

func handleRequestItemConsume(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestItemConsumeBody]) {
	if c.Type != consumable2.CommandRequestItemConsume {
		return
	}
	err := consumable.NewProcessor(l, ctx).RequestItemConsume(byte(c.WorldId), byte(c.ChannelId), uint32(c.CharacterId), int16(c.Body.Source), c.Body.ItemId, c.Body.Quantity)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to consume item in slot [%d] as expected.", c.CharacterId, c.Body.Source)
	}
}

func handleRequestScroll(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestScrollBody]) {
	if c.Type != consumable2.CommandRequestScroll {
		return
	}
	err := consumable.NewProcessor(l, ctx).RequestScroll(uint32(c.CharacterId), int16(c.Body.ScrollSlot), int16(c.Body.EquipSlot), c.Body.WhiteScroll, c.Body.LegendarySpirit)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to use scroll in slot [%d] as expected.", c.CharacterId, c.Body.ScrollSlot)
	}
}

func handleApplyConsumableEffect(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.ApplyConsumableEffectBody]) {
	if c.Type != consumable2.CommandApplyConsumableEffect {
		return
	}
	err := consumable.NewProcessor(l, ctx).ApplyConsumableEffect(c.TransactionId, byte(c.WorldId), byte(c.ChannelId), uint32(c.CharacterId), c.Body.ItemId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to apply consumable effect [%d] as expected.", c.CharacterId, c.Body.ItemId)
	}
}
