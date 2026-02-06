package instance_transport

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopic           = "EVENT_TOPIC_INSTANCE_TRANSPORT"
	EventTypeTransitEntered = "TRANSIT_ENTERED"
)

type Event[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type TransitEnteredEventBody struct {
	RouteId         uuid.UUID  `json:"routeId"`
	InstanceId      uuid.UUID  `json:"instanceId"`
	ChannelId       channel.Id `json:"channelId"`
	DurationSeconds uint32     `json:"durationSeconds"`
	Message         string     `json:"message"`
}
