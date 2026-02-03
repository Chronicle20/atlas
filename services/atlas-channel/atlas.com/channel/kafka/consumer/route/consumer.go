package route

import (
	consumer2 "atlas-channel/kafka/consumer"
	route2 "atlas-channel/kafka/message/route"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitConsumers initializes the route status event consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("route_status_event")(route2.EnvEventTopicStatus)(consumerGroupId), 
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), 
				consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers initializes the route status event handlers
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(route2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventArrived(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeparted(sc, wp))))
			}
		}
	}
}

// handleStatusEventArrived handles ARRIVED events
func handleStatusEventArrived(sc server.Model, wp writer.Producer) message.Handler[route2.StatusEvent[route2.ArrivedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e route2.StatusEvent[route2.ArrivedStatusEventBody]) {
		if e.Type != route2.EventStatusArrived {
			return
		}

		t := tenant.MustFromContext(ctx)
		mapId := _map2.Id(e.Body.MapId)

		l.Debugf("Transport route [%s] has arrived at map [%d].", e.RouteId, mapId)

		// Broadcast to all characters in the map
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(mapId, uuid.Nil),
			session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l, t)(writer.TransportStateEnter1, false)))

		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast transport arrival to characters in map [%d].", mapId)
		}
	}
}

// handleStatusEventDeparted handles DEPARTED events
func handleStatusEventDeparted(sc server.Model, wp writer.Producer) message.Handler[route2.StatusEvent[route2.DepartedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e route2.StatusEvent[route2.DepartedStatusEventBody]) {
		if e.Type != route2.EventStatusDeparted {
			return
		}

		t := tenant.MustFromContext(ctx)
		mapId := _map2.Id(e.Body.MapId)

		l.Debugf("Transport route [%s] has departed from map [%d].", e.RouteId, mapId)

		// Broadcast to all characters in the map
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(mapId, uuid.Nil),
			session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l, t)(writer.TransportStateMove1, false)))

		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast transport departure to characters in map [%d].", mapId)
		}
	}
}
