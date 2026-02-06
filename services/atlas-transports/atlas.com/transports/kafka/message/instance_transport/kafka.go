package instance_transport

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_INSTANCE_TRANSPORT"
	CommandStart    = "START"
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
	EnvEventTopic           = "EVENT_TOPIC_INSTANCE_TRANSPORT"
	EventTypeStarted        = "STARTED"
	EventTypeTransitEntered = "TRANSIT_ENTERED"
	EventTypeCompleted      = "COMPLETED"
	EventTypeCancelled      = "CANCELLED"

	CancelReasonMapExit = "MAP_EXIT"
	CancelReasonLogout  = "LOGOUT"
	CancelReasonStuck   = "STUCK"
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
