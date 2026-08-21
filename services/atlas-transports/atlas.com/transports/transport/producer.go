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

// voyageStatusEventProvider keys on the voyage id so every event for one voyage
// lands on one partition — VOYAGE_ARRIVED can therefore never overtake the
// VOYAGE_DEPARTED of the same voyage.
func voyageStatusEventProvider(theType string, body transport.VoyageStatusEventBody, routeId uuid.UUID) model.Provider[[]kafka.Message] {
	value := transport.StatusEvent[transport.VoyageStatusEventBody]{
		RouteId: routeId,
		Type:    theType,
		Body:    body,
	}
	return producer.SingleMessageProvider([]byte(body.VoyageId.String()), value)
}

func VoyageDepartedStatusEventProvider(routeId uuid.UUID, body transport.VoyageStatusEventBody) model.Provider[[]kafka.Message] {
	return voyageStatusEventProvider(transport.EventStatusVoyageDeparted, body, routeId)
}

func VoyageArrivedStatusEventProvider(routeId uuid.UUID, body transport.VoyageStatusEventBody) model.Provider[[]kafka.Message] {
	return voyageStatusEventProvider(transport.EventStatusVoyageArrived, body, routeId)
}
