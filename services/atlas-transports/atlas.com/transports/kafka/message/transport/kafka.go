package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus = "EVENT_TOPIC_TRANSPORT_STATUS"
	EventStatusArrived  = "ARRIVED"
	EventStatusDeparted = "DEPARTED"
)

type StatusEvent[E any] struct {
	RouteId uuid.UUID `json:"routeId"`
	Type    string    `json:"type"`
	Body    E         `json:"body"`
}

type ArrivedStatusEventBody struct {
	MapId _map.Id `json:"mapId"`
}

type DepartedStatusEventBody struct {
	MapId _map.Id `json:"mapId"`
}
