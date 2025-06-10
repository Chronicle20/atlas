package compartment

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/compartment"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("compartment_status_event")(compartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentItemReservationCancelledEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentMergeCompleteEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentSortCompleteEvent(sc, wp))))
			}
		}
	}
}

func handleCompartmentItemReservationCancelledEvent(sc server.Model, wp writer.Producer) message.Handler[compartment.StatusEvent[compartment.ReservationCancelledEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReservationCancelledEventBody]) {
		if e.Type != compartment.StatusEventTypeReservationCancelled {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true)))
	}
}

func handleCompartmentMergeCompleteEvent(sc server.Model, wp writer.Producer) message.Handler[compartment.StatusEvent[compartment.MergeCompleteEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.MergeCompleteEventBody]) {
		if e.Type != compartment.StatusEventTypeMergeComplete {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.CompartmentMerge)(writer.CompartmentMergeBody(e.Body.Type)))
	}
}

func handleCompartmentSortCompleteEvent(sc server.Model, wp writer.Producer) message.Handler[compartment.StatusEvent[compartment.SortCompleteEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.SortCompleteEventBody]) {
		if e.Type != compartment.StatusEventTypeSortComplete {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.CompartmentSort)(writer.CompartmentSortBody(e.Body.Type)))
	}
}
