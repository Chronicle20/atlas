package transport

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicStatus = "EVENT_TOPIC_TRANSPORT_STATUS"
	EventStatusArrived  = "ARRIVED"
	EventStatusDeparted = "DEPARTED"

	// Voyage lifecycle, distinct from the observation-deck visuals above.
	// ARRIVED/DEPARTED tell a watcher on the pier what the docked ship looks
	// like and carry only a map id; these name a concrete trip in a concrete
	// channel and carry its whole scope (FR-V3, FR-V4).
	EventStatusVoyageDeparted = "VOYAGE_DEPARTED"
	EventStatusVoyageArrived  = "VOYAGE_ARRIVED"
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

// VoyageStatusEventBody serves both voyage types; the envelope's Type
// discriminates. VOYAGE_ARRIVED carries DepartedAt too, so a consumer can
// compute voyage duration without holding the departure event.
type VoyageStatusEventBody struct {
	VoyageId         uuid.UUID  `json:"voyageId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	StagingMapId     _map.Id    `json:"stagingMapId"`
	EnRouteMapIds    []_map.Id  `json:"enRouteMapIds"`
	DestinationMapId _map.Id    `json:"destinationMapId"`
	ObservationMapId _map.Id    `json:"observationMapId"`
	DepartedAt       time.Time  `json:"departedAt"`
}
