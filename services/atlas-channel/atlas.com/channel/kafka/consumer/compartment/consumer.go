package compartment

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/compartment"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	invpkt "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentItemReservationCancelledEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentMergeCompleteEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentSortCompleteEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode))
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := enableActions(l)(ctx)(wp)(s)
			if err != nil {
				return err
			}
			err = session.Announce(l)(ctx)(wp)(invpkt.CompartmentMergeWriter)(invpkt.NewCompartmentMerge(e.Body.Type).Encode)(s)
			if err != nil {
				return err
			}
			return nil
		})
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := enableActions(l)(ctx)(wp)(s)
			if err != nil {
				return err
			}
			err = session.Announce(l)(ctx)(wp)(invpkt.CompartmentSortWriter)(invpkt.NewCompartmentSort(e.Body.Type).Encode)(s)
			if err != nil {
				return err
			}
			return nil
		})
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)
		}
	}
}
