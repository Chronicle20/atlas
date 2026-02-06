package transport

import (
	"errors"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

// Builder is a builder for Model
type Builder struct {
	id                     uuid.UUID
	name                   string
	startMapId             _map.Id
	stagingMapId           _map.Id
	enRouteMapIds          []_map.Id
	destinationMapId       _map.Id
	observationMapId       _map.Id
	state                  RouteState
	schedule               []TripScheduleModel
	boardingWindowDuration time.Duration
	preDepartureDuration   time.Duration
	travelDuration         time.Duration
	cycleInterval          time.Duration
}

// NewBuilder creates a new builder for Model
func NewBuilder(name string) *Builder {
	return &Builder{
		id:            uuid.New(),
		name:          name,
		enRouteMapIds: []_map.Id{},
		state:         OutOfService,
		schedule:      []TripScheduleModel{},
	}
}

// SetId sets the route ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetName sets the route name
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetStartMapId sets the starting map ID
func (b *Builder) SetStartMapId(startMapId _map.Id) *Builder {
	b.startMapId = startMapId
	return b
}

// SetStagingMapId sets the staging map ID
func (b *Builder) SetStagingMapId(stagingMapId _map.Id) *Builder {
	b.stagingMapId = stagingMapId
	return b
}

// SetEnRouteMapIds sets the en-route map IDs
func (b *Builder) SetEnRouteMapIds(enRouteMapIds []_map.Id) *Builder {
	b.enRouteMapIds = enRouteMapIds
	return b
}

// AddEnRouteMapId adds an en-route map ID
func (b *Builder) AddEnRouteMapId(enRouteMapId _map.Id) *Builder {
	b.enRouteMapIds = append(b.enRouteMapIds, enRouteMapId)
	return b
}

// SetDestinationMapId sets the destination map ID
func (b *Builder) SetDestinationMapId(destinationMapId _map.Id) *Builder {
	b.destinationMapId = destinationMapId
	return b
}

// SetObservationMapId sets the observation map ID
func (b *Builder) SetObservationMapId(observationMapId _map.Id) *Builder {
	b.observationMapId = observationMapId
	return b
}

// SetBoardingWindowDuration sets the boarding window duration
func (b *Builder) SetBoardingWindowDuration(boardingWindowDuration time.Duration) *Builder {
	b.boardingWindowDuration = boardingWindowDuration
	return b
}

// SetPreDepartureDuration sets the pre-departure duration
func (b *Builder) SetPreDepartureDuration(preDepartureDuration time.Duration) *Builder {
	b.preDepartureDuration = preDepartureDuration
	return b
}

// SetTravelDuration sets the travel duration
func (b *Builder) SetTravelDuration(travelDuration time.Duration) *Builder {
	b.travelDuration = travelDuration
	return b
}

// SetCycleInterval sets the cycle interval
func (b *Builder) SetCycleInterval(cycleInterval time.Duration) *Builder {
	b.cycleInterval = cycleInterval
	return b
}

func (b *Builder) SetState(state RouteState) *Builder {
	b.state = state
	return b
}

func (b *Builder) SetSchedule(schedule []TripScheduleModel) *Builder {
	b.schedule = schedule
	return b
}

func (b *Builder) AddToSchedule(schedule TripScheduleModel) *Builder {
	b.schedule = append(b.schedule, schedule)
	return b
}

// Build builds the Model with validation
func (b *Builder) Build() (Model, error) {
	if b.name == "" {
		return Model{}, errors.New("route name must not be empty")
	}
	if len(b.enRouteMapIds) == 0 {
		return Model{}, errors.New("at least one en-route map ID is required")
	}
	if b.boardingWindowDuration <= 0 {
		return Model{}, errors.New("boarding window duration must be positive")
	}
	if b.preDepartureDuration < 0 {
		return Model{}, errors.New("pre-departure duration must not be negative")
	}
	if b.travelDuration <= 0 {
		return Model{}, errors.New("travel duration must be positive")
	}
	if b.cycleInterval <= 0 {
		return Model{}, errors.New("cycle interval must be positive")
	}

	return Model{
		id:                     b.id,
		name:                   b.name,
		startMapId:             b.startMapId,
		stagingMapId:           b.stagingMapId,
		enRouteMapIds:          b.enRouteMapIds,
		destinationMapId:       b.destinationMapId,
		observationMapId:       b.observationMapId,
		state:                  b.state,
		schedule:               b.schedule,
		boardingWindowDuration: b.boardingWindowDuration,
		preDepartureDuration:   b.preDepartureDuration,
		travelDuration:         b.travelDuration,
		cycleInterval:          b.cycleInterval,
	}, nil
}

// SharedVesselBuilder is a builder for SharedVesselModel
type SharedVesselBuilder struct {
	id              string
	name            string
	routeAID        string
	routeBID        string
	turnaroundDelay time.Duration
}

// NewSharedVesselBuilder creates a new builder for SharedVesselModel
func NewSharedVesselBuilder() *SharedVesselBuilder {
	return &SharedVesselBuilder{}
}

// SetId sets the shared vessel ID
func (b *SharedVesselBuilder) SetId(id string) *SharedVesselBuilder {
	b.id = id
	return b
}

func (b *SharedVesselBuilder) SetName(name string) *SharedVesselBuilder {
	b.name = name
	return b
}

// SetRouteAID sets the ID of route A
func (b *SharedVesselBuilder) SetRouteAID(routeAID string) *SharedVesselBuilder {
	b.routeAID = routeAID
	return b
}

// SetRouteBID sets the ID of route B
func (b *SharedVesselBuilder) SetRouteBID(routeBID string) *SharedVesselBuilder {
	b.routeBID = routeBID
	return b
}

// SetTurnaroundDelay sets the turnaround delay
func (b *SharedVesselBuilder) SetTurnaroundDelay(turnaroundDelay time.Duration) *SharedVesselBuilder {
	b.turnaroundDelay = turnaroundDelay
	return b
}

// Build builds the SharedVesselModel with validation
func (b *SharedVesselBuilder) Build() (SharedVesselModel, error) {
	if b.routeAID == "" {
		return SharedVesselModel{}, errors.New("route A ID must not be empty")
	}
	if b.routeBID == "" {
		return SharedVesselModel{}, errors.New("route B ID must not be empty")
	}
	if b.turnaroundDelay <= 0 {
		return SharedVesselModel{}, errors.New("turnaround delay must be positive")
	}

	return NewSharedVesselModel(
		b.id,
		b.name,
		b.routeAID,
		b.routeBID,
		b.turnaroundDelay,
	), nil
}

// TripScheduleBuilder is a builder for TripScheduleModel
type TripScheduleBuilder struct {
	tripId         uuid.UUID
	routeId        uuid.UUID
	boardingOpen   time.Time
	boardingClosed time.Time
	departure      time.Time
	arrival        time.Time
}

// NewTripScheduleBuilder creates a new builder for TripScheduleModel
func NewTripScheduleBuilder() *TripScheduleBuilder {
	return &TripScheduleBuilder{
		tripId: uuid.New(),
	}
}

// SetTripId sets the trip ID
func (b *TripScheduleBuilder) SetTripId(tripId uuid.UUID) *TripScheduleBuilder {
	b.tripId = tripId
	return b
}

// SetRouteId sets the route ID
func (b *TripScheduleBuilder) SetRouteId(routeId uuid.UUID) *TripScheduleBuilder {
	b.routeId = routeId
	return b
}

// SetBoardingOpen sets the boarding open time
func (b *TripScheduleBuilder) SetBoardingOpen(boardingOpen time.Time) *TripScheduleBuilder {
	b.boardingOpen = boardingOpen
	return b
}

// SetBoardingClosed sets the boarding closed time
func (b *TripScheduleBuilder) SetBoardingClosed(boardingClosed time.Time) *TripScheduleBuilder {
	b.boardingClosed = boardingClosed
	return b
}

// SetDeparture sets the departure time
func (b *TripScheduleBuilder) SetDeparture(departure time.Time) *TripScheduleBuilder {
	b.departure = departure
	return b
}

// SetArrival sets the arrival time
func (b *TripScheduleBuilder) SetArrival(arrival time.Time) *TripScheduleBuilder {
	b.arrival = arrival
	return b
}

// Build builds the TripScheduleModel with validation
func (b *TripScheduleBuilder) Build() (TripScheduleModel, error) {
	if b.routeId == uuid.Nil {
		return TripScheduleModel{}, errors.New("route ID must not be nil")
	}
	if !b.boardingOpen.Before(b.boardingClosed) {
		return TripScheduleModel{}, errors.New("boarding open must be before boarding closed")
	}
	if b.departure.Before(b.boardingClosed) {
		return TripScheduleModel{}, errors.New("departure must not be before boarding closed")
	}
	if !b.departure.Before(b.arrival) {
		return TripScheduleModel{}, errors.New("departure must be before arrival")
	}

	return NewTripScheduleModel(
		b.tripId,
		b.routeId,
		b.boardingOpen,
		b.boardingClosed,
		b.departure,
		b.arrival,
	), nil
}
