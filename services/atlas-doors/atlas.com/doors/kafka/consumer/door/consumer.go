package door

import (
	consumer2 "atlas-doors/kafka/consumer"
	enginedoor "atlas-doors/door"
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
			rf(consumer2.NewConfig(l)("door_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawn))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemove))); err != nil {
			return err
		}
		return nil
	}
}

func handleSpawn(l logrus.FieldLogger, ctx context.Context, c Command[SpawnBody]) {
	if c.Type != CommandTypeSpawn {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, err := enginedoor.NewProcessor(l, ctx).Spawn(f, c.OwnerCharacterId, c.Body.SkillId, c.Body.SkillLevel, c.Body.X, c.Body.Y)
	if err != nil {
		l.WithError(err).Debugf("door spawn rejected for character %d", c.OwnerCharacterId)
	}
}

func handleRemove(l logrus.FieldLogger, ctx context.Context, c Command[RemoveBody]) {
	if c.Type != CommandTypeRemove {
		return
	}
	reason := c.Body.Reason
	if reason == "" {
		reason = enginedoor.RemoveReasonRecast
	}
	_ = enginedoor.NewProcessor(l, ctx).RemoveByOwner(c.OwnerCharacterId, reason)
}
