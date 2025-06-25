package route

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus = "EVENT_TOPIC_TRANSPORT_STATUS"
	EventStatusArrived  = "ARRIVED"
	EventStatusDeparted = "DEPARTED"
)

// StatusEvent is a generic event for transport route status changes
type StatusEvent[E any] struct {
	RouteId uuid.UUID `json:"routeId"`
	Type    string    `json:"type"`
	Body    E         `json:"body"`
}

// ArrivedStatusEventBody is the body for ARRIVED events
type ArrivedStatusEventBody struct {
	MapId _map.Id `json:"mapId"`
}

// DepartedStatusEventBody is the body for DEPARTED events
type DepartedStatusEventBody struct {
	MapId _map.Id `json:"mapId"`
}