package instance_transport

import (
	consumer2 "atlas-channel/kafka/consumer"
	it "atlas-channel/kafka/message/instance_transport"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitConsumers initializes the instance transport event consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("instance_transport_event")(it.EnvEventTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers initializes the instance transport event handlers
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(it.EnvEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleTransitEnteredEvent(sc, wp))))
			}
		}
	}
}

// handleTransitEnteredEvent handles TRANSIT_ENTERED events to display clock and optional message
func handleTransitEnteredEvent(sc server.Model, wp writer.Producer) message.Handler[it.Event[it.TransitEnteredEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e it.Event[it.TransitEnteredEventBody]) {
		if e.Type != it.EventTypeTransitEntered {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !sc.Is(t, e.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Character [%d] entered transit, showing clock for [%d] seconds.", e.CharacterId, e.Body.DurationSeconds)

		// Send CLOCK packet
		duration := time.Duration(e.Body.DurationSeconds) * time.Second
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.Clock)(writer.TimerClockBody(l, t)(duration)))
		if err != nil {
			l.WithError(err).Errorf("Unable to send clock to character [%d].", e.CharacterId)
		}

		// Send ScriptProgress packet if message is present
		if e.Body.Message != "" {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
				session.Announce(l)(ctx)(wp)(writer.ScriptProgress)(writer.ScriptProgressBody(e.Body.Message)))
			if err != nil {
				l.WithError(err).Errorf("Unable to send script progress to character [%d].", e.CharacterId)
			}
		}
	}
}
