package mist

import (
	consumer2 "atlas-maps/kafka/consumer"
	mistKafka "atlas-maps/kafka/message/mist"
	"atlas-maps/kafka/producer"
	mistDomain "atlas-maps/mist"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// processorFactory builds a mist.Processor for a given context. Defaults to
// the canonical implementation; tests override it to inject a recording
// producer or fake registry.
var processorFactory = func(l logrus.FieldLogger, ctx context.Context) mistDomain.Processor {
	return mistDomain.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
}

// InitConsumers registers the mist command consumer on the shared
// ConsumerManagerFactory. It mirrors the curry shape of the sibling map
// command consumer so it slots into main.go's wiring identically.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mist_command")(mistKafka.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

// InitHandlers attaches the typed handlers for MIST_CREATE and MIST_CANCEL.
// One handler per command type, each filtering on the envelope Type, mirroring
// the in-tree skill command consumer pattern.
func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(mistKafka.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelCommand()))); err != nil {
			return err
		}
		return nil
	}
}

// handleCreateCommand dispatches a MIST_CREATE envelope to the mist Processor.
// The processor handles registry insertion and MIST_CREATED emission.
func handleCreateCommand() func(l logrus.FieldLogger, ctx context.Context, c mistKafka.Command[mistKafka.CreateCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mistKafka.Command[mistKafka.CreateCommandBody]) {
		if c.Type != mistKafka.CommandTypeCreate {
			return
		}

		l.Debugf("Received MIST_CREATE for owner [%s/%d] map [%d] instance [%s] disease [%s] duration [%d]ms.",
			c.Body.OwnerType, c.Body.OwnerId, c.Body.MapId, c.Body.Instance, c.Body.Disease, c.Body.Duration)

		if _, err := processorFactory(l, ctx).Create(c.Body); err != nil {
			l.WithError(err).Errorf("Unable to create mist for owner [%s/%d] on map [%d] instance [%s].",
				c.Body.OwnerType, c.Body.OwnerId, c.Body.MapId, c.Body.Instance)
		}
	}
}

// handleCancelCommand dispatches a MIST_CANCEL envelope to the mist Processor.
// The processor removes the mist and emits MIST_DESTROYED with reason CANCELLED.
func handleCancelCommand() func(l logrus.FieldLogger, ctx context.Context, c mistKafka.Command[mistKafka.CancelCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mistKafka.Command[mistKafka.CancelCommandBody]) {
		if c.Type != mistKafka.CommandTypeCancel {
			return
		}

		l.Debugf("Received MIST_CANCEL for mist [%s].", c.Body.MistId)

		if _, err := processorFactory(l, ctx).Destroy(c.Body.MistId, mistKafka.ReasonCancelled); err != nil {
			l.WithError(err).Errorf("Unable to cancel mist [%s].", c.Body.MistId)
		}
	}
}
