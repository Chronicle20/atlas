// Package transport mirrors the atlas-transports status events this service
// consumes (source of truth:
// services/atlas-transports/atlas.com/transports/kafka/message/transport/kafka.go).
// Only the consumed types/fields are mirrored; unknown event types on the
// topic are ignored by the handlers' type guards.
package transport

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_TRANSPORT_STATUS"
)

const (

	// EventStatusVoyageDeparted names a concrete trip in a concrete channel
	// and carries its whole scope (FR-V3, FR-V4).
	EventStatusVoyageDeparted = "VOYAGE_DEPARTED"

	// EventStatusVoyageArrived is the terminal status event for a voyage
	// (source of truth:
	// services/atlas-transports/atlas.com/transports/kafka/message/transport/kafka.go).
	// Consumed by events/crimsonbalrog's ArrivalProcessor (Task 27) to
	// complete any still-ACTIVE occurrence scoped to the arriving voyage.
	EventStatusVoyageArrived = "VOYAGE_ARRIVED"
)

// StatusEvent is the envelope every transport status event arrives in.
type StatusEvent[E any] struct {
	RouteId uuid.UUID `json:"routeId"`
	Type    string    `json:"type"`
	Body    E         `json:"body"`
}

// VoyageStatusEventBody carries the full voyage scope a VOYAGE_DEPARTED (or
// VOYAGE_ARRIVED) event describes.
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
