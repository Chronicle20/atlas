package kite

import (
	consumer2 "atlas-kites/kafka/consumer"
	kiteMsg "atlas-kites/kafka/message/kite"
	"atlas-kites/kite"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("kite_command")(kiteMsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(kiteMsg.EnvCommandTopic)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c kiteMsg.Command[kiteMsg.CreateCommandBody]) {
	if c.Type != kiteMsg.CommandKiteCreate {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	if _, err := kite.NewProcessor(l, ctx).CreateAndEmit(f, c.CharacterId, c.Body); err != nil {
		l.WithError(err).Debugf("Unable to create kite for character [%d].", c.CharacterId)
	}
}

func handleDestroyCommand(l logrus.FieldLogger, ctx context.Context, c kiteMsg.Command[kiteMsg.DestroyCommandBody]) {
	if c.Type != kiteMsg.CommandKiteDestroy {
		return
	}
	// The Task 6 contract defines only the two teardown reasons
	// (OWNER_LEFT, OWNER_LOGGED_OUT); no producer for this DESTROY command
	// exists yet anywhere in the plan, so there is no real cause to report.
	// OWNER_LEFT is the closest available value -- revisit this once an
	// actual command producer (e.g. a player-initiated pickup) is wired.
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(c.CharacterId, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for character [%d].", c.CharacterId)
	}
}
