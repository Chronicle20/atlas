package transport

import (
	"atlas-transports/kafka/message/transport"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func ArrivedStatusEventProvider(routeId uuid.UUID, mapId _map.Id) model.Provider[[]kafka.Message] {
	value := transport.StatusEvent[transport.ArrivedStatusEventBody]{
		RouteId: routeId,
		Type:    transport.EventStatusArrived,
		Body: transport.ArrivedStatusEventBody{
			MapId: mapId,
		},
	}
	return producer.SingleMessageProvider([]byte(routeId.String()), value)
}

func DepartedStatusEventProvider(routeId uuid.UUID, mapId _map.Id) model.Provider[[]kafka.Message] {
	value := transport.StatusEvent[transport.DepartedStatusEventBody]{
		RouteId: routeId,
		Type:    transport.EventStatusDeparted,
		Body: transport.DepartedStatusEventBody{
			MapId: mapId,
		},
	}
	return producer.SingleMessageProvider([]byte(routeId.String()), value)
}
