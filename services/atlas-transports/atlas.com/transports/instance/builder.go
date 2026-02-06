package instance

import (
	"errors"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

type RouteBuilder struct {
	id               uuid.UUID
	name             string
	startMapId       _map.Id
	transitMapIds    []_map.Id
	destinationMapId _map.Id
	capacity         uint32
	boardingWindow   time.Duration
	travelDuration   time.Duration
	transitMessage   string
}

func NewRouteBuilder(name string) *RouteBuilder {
	return &RouteBuilder{
		id:   uuid.New(),
		name: name,
	}
}

func (b *RouteBuilder) SetId(id uuid.UUID) *RouteBuilder {
	b.id = id
	return b
}

func (b *RouteBuilder) SetStartMapId(startMapId _map.Id) *RouteBuilder {
	b.startMapId = startMapId
	return b
}

func (b *RouteBuilder) SetTransitMapIds(transitMapIds []_map.Id) *RouteBuilder {
	b.transitMapIds = transitMapIds
	return b
}

func (b *RouteBuilder) SetDestinationMapId(destinationMapId _map.Id) *RouteBuilder {
	b.destinationMapId = destinationMapId
	return b
}

func (b *RouteBuilder) SetCapacity(capacity uint32) *RouteBuilder {
	b.capacity = capacity
	return b
}

func (b *RouteBuilder) SetBoardingWindow(boardingWindow time.Duration) *RouteBuilder {
	b.boardingWindow = boardingWindow
	return b
}

func (b *RouteBuilder) SetTravelDuration(travelDuration time.Duration) *RouteBuilder {
	b.travelDuration = travelDuration
	return b
}

func (b *RouteBuilder) SetTransitMessage(transitMessage string) *RouteBuilder {
	b.transitMessage = transitMessage
	return b
}

func (b *RouteBuilder) Build() (RouteModel, error) {
	if b.name == "" {
		return RouteModel{}, errors.New("route name must not be empty")
	}
	if b.capacity == 0 {
		return RouteModel{}, errors.New("capacity must be greater than zero")
	}
	if b.boardingWindow <= 0 {
		return RouteModel{}, errors.New("boarding window must be positive")
	}
	if b.travelDuration < 0 {
		return RouteModel{}, errors.New("travel duration must not be negative")
	}
	if len(b.transitMapIds) == 0 {
		return RouteModel{}, errors.New("transit map ids must not be empty")
	}

	return RouteModel{
		id:               b.id,
		name:             b.name,
		startMapId:       b.startMapId,
		transitMapIds:    b.transitMapIds,
		destinationMapId: b.destinationMapId,
		capacity:         b.capacity,
		boardingWindow:   b.boardingWindow,
		travelDuration:   b.travelDuration,
		transitMessage:   b.transitMessage,
	}, nil
}
