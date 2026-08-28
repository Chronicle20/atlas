package instance_transport

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_INSTANCE_TRANSPORT"
)

const (
	CommandStart = "START"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StartCommandBody struct {
	RouteId   uuid.UUID  `json:"routeId"`
	ChannelId channel.Id `json:"channelId"`
}

const (
	EnvEventTopic topic.Token = "EVENT_TOPIC_INSTANCE_TRANSPORT"
)

const (
	EventTypeStarted        = "STARTED"
	EventTypeTransitEntered = "TRANSIT_ENTERED"
	EventTypeCompleted      = "COMPLETED"
	EventTypeCancelled      = "CANCELLED"

	CancelReasonMapExit = "MAP_EXIT"
	CancelReasonLogout  = "LOGOUT"
	CancelReasonStuck   = "STUCK"
	// CancelReasonTimeout is emitted when the travel timer expires on a route
	// that declares a forced return. The character did not complete the trip —
	// the client's own map data (timeLimit + forcedReturn) treats running out
	// of flight time as a failure that sends them back where they started.
	CancelReasonTimeout = "TIMEOUT"
)

type Event[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StartedEventBody struct {
	RouteId    uuid.UUID `json:"routeId"`
	InstanceId uuid.UUID `json:"instanceId"`
}

type TransitEnteredEventBody struct {
	RouteId         uuid.UUID  `json:"routeId"`
	InstanceId      uuid.UUID  `json:"instanceId"`
	ChannelId       channel.Id `json:"channelId"`
	DurationSeconds uint32     `json:"durationSeconds"`
	Message         string     `json:"message"`
}

type CompletedEventBody struct {
	RouteId    uuid.UUID `json:"routeId"`
	InstanceId uuid.UUID `json:"instanceId"`
}

type CancelledEventBody struct {
	RouteId    uuid.UUID `json:"routeId"`
	InstanceId uuid.UUID `json:"instanceId"`
	Reason     string    `json:"reason"`
}
