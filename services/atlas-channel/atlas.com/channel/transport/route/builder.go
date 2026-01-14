package route

import (
	"errors"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"time"
)

var (
	ErrInvalidId = errors.New("route id must not be nil")
)

// modelBuilder is a builder for Model
type modelBuilder struct {
	id                     uuid.UUID
	name                   string
	startMapId             _map.Id
	stagingMapId           _map.Id
	enRouteMapIds          []_map.Id
	destinationMapId       _map.Id
	state                  RouteState
	schedule               []TripScheduleModel
	boardingWindowDuration time.Duration
	preDepartureDuration   time.Duration
	travelDuration         time.Duration
	cycleInterval          time.Duration
}

// NewModelBuilder creates a new builder for Model
func NewModelBuilder(name string) *modelBuilder {
	return &modelBuilder{
		id:            uuid.New(),
		name:          name,
		enRouteMapIds: []_map.Id{},
		state:         OutOfService,
		schedule:      []TripScheduleModel{},
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder(name string) *modelBuilder {
	return NewModelBuilder(name)
}

// CloneModel creates a builder from an existing Model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:                     m.id,
		name:                   m.name,
		startMapId:             m.startMapId,
		stagingMapId:           m.stagingMapId,
		enRouteMapIds:          m.enRouteMapIds,
		destinationMapId:       m.destinationMapId,
		state:                  m.state,
		schedule:               m.schedule,
		boardingWindowDuration: m.boardingWindowDuration,
		preDepartureDuration:   m.preDepartureDuration,
		travelDuration:         m.travelDuration,
		cycleInterval:          m.cycleInterval,
	}
}

func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetName(name string) *modelBuilder {
	b.name = name
	return b
}

func (b *modelBuilder) SetStartMapId(startMapId _map.Id) *modelBuilder {
	b.startMapId = startMapId
	return b
}

func (b *modelBuilder) SetStagingMapId(stagingMapId _map.Id) *modelBuilder {
	b.stagingMapId = stagingMapId
	return b
}

func (b *modelBuilder) SetEnRouteMapIds(enRouteMapIds []_map.Id) *modelBuilder {
	b.enRouteMapIds = enRouteMapIds
	return b
}

func (b *modelBuilder) AddEnRouteMapId(enRouteMapId _map.Id) *modelBuilder {
	b.enRouteMapIds = append(b.enRouteMapIds, enRouteMapId)
	return b
}

func (b *modelBuilder) SetDestinationMapId(destinationMapId _map.Id) *modelBuilder {
	b.destinationMapId = destinationMapId
	return b
}

func (b *modelBuilder) SetBoardingWindowDuration(boardingWindowDuration time.Duration) *modelBuilder {
	b.boardingWindowDuration = boardingWindowDuration
	return b
}

func (b *modelBuilder) SetPreDepartureDuration(preDepartureDuration time.Duration) *modelBuilder {
	b.preDepartureDuration = preDepartureDuration
	return b
}

func (b *modelBuilder) SetTravelDuration(travelDuration time.Duration) *modelBuilder {
	b.travelDuration = travelDuration
	return b
}

func (b *modelBuilder) SetCycleInterval(cycleInterval time.Duration) *modelBuilder {
	b.cycleInterval = cycleInterval
	return b
}

func (b *modelBuilder) SetState(state RouteState) *modelBuilder {
	b.state = state
	return b
}

func (b *modelBuilder) SetSchedule(schedule []TripScheduleModel) *modelBuilder {
	b.schedule = schedule
	return b
}

func (b *modelBuilder) AddToSchedule(schedule TripScheduleModel) *modelBuilder {
	b.schedule = append(b.schedule, schedule)
	return b
}

func (b *modelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:                     b.id,
		name:                   b.name,
		startMapId:             b.startMapId,
		stagingMapId:           b.stagingMapId,
		enRouteMapIds:          b.enRouteMapIds,
		destinationMapId:       b.destinationMapId,
		state:                  b.state,
		schedule:               b.schedule,
		boardingWindowDuration: b.boardingWindowDuration,
		preDepartureDuration:   b.preDepartureDuration,
		travelDuration:         b.travelDuration,
		cycleInterval:          b.cycleInterval,
	}, nil
}

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// sharedVesselBuilder is a builder for SharedVesselModel
type sharedVesselBuilder struct {
	id              string
	routeAID        uuid.UUID
	routeBID        uuid.UUID
	turnaroundDelay time.Duration
}

// NewSharedVesselModelBuilder creates a new builder for SharedVesselModel
func NewSharedVesselModelBuilder() *sharedVesselBuilder {
	return &sharedVesselBuilder{
		id: uuid.New().String(),
	}
}

// NewSharedVesselBuilder is an alias for backward compatibility
func NewSharedVesselBuilder() *sharedVesselBuilder {
	return NewSharedVesselModelBuilder()
}

func (b *sharedVesselBuilder) SetId(id string) *sharedVesselBuilder {
	b.id = id
	return b
}

func (b *sharedVesselBuilder) SetRouteAID(routeAID uuid.UUID) *sharedVesselBuilder {
	b.routeAID = routeAID
	return b
}

func (b *sharedVesselBuilder) SetRouteBID(routeBID uuid.UUID) *sharedVesselBuilder {
	b.routeBID = routeBID
	return b
}

func (b *sharedVesselBuilder) SetTurnaroundDelay(turnaroundDelay time.Duration) *sharedVesselBuilder {
	b.turnaroundDelay = turnaroundDelay
	return b
}

func (b *sharedVesselBuilder) Build() (SharedVesselModel, error) {
	return SharedVesselModel{
		id:              b.id,
		routeAID:        b.routeAID,
		routeBID:        b.routeBID,
		turnaroundDelay: b.turnaroundDelay,
	}, nil
}

func (b *sharedVesselBuilder) MustBuild() SharedVesselModel {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// tripScheduleBuilder is a builder for TripScheduleModel
type tripScheduleBuilder struct {
	tripId         uuid.UUID
	routeId        uuid.UUID
	boardingOpen   time.Time
	boardingClosed time.Time
	departure      time.Time
	arrival        time.Time
}

// NewTripScheduleModelBuilder creates a new builder for TripScheduleModel
func NewTripScheduleModelBuilder() *tripScheduleBuilder {
	return &tripScheduleBuilder{
		tripId: uuid.New(),
	}
}

// CloneTripSchedule creates a builder from an existing TripScheduleModel
func CloneTripSchedule(m TripScheduleModel) *tripScheduleBuilder {
	return &tripScheduleBuilder{
		tripId:         m.tripId,
		routeId:        m.routeId,
		boardingOpen:   m.boardingOpen,
		boardingClosed: m.boardingClosed,
		departure:      m.departure,
		arrival:        m.arrival,
	}
}

// NewTripScheduleBuilder is an alias for backward compatibility
func NewTripScheduleBuilder() *tripScheduleBuilder {
	return NewTripScheduleModelBuilder()
}

func (b *tripScheduleBuilder) SetTripId(tripId uuid.UUID) *tripScheduleBuilder {
	b.tripId = tripId
	return b
}

func (b *tripScheduleBuilder) SetRouteId(routeId uuid.UUID) *tripScheduleBuilder {
	b.routeId = routeId
	return b
}

func (b *tripScheduleBuilder) SetBoardingOpen(boardingOpen time.Time) *tripScheduleBuilder {
	b.boardingOpen = boardingOpen
	return b
}

func (b *tripScheduleBuilder) SetBoardingClosed(boardingClosed time.Time) *tripScheduleBuilder {
	b.boardingClosed = boardingClosed
	return b
}

func (b *tripScheduleBuilder) SetDeparture(departure time.Time) *tripScheduleBuilder {
	b.departure = departure
	return b
}

func (b *tripScheduleBuilder) SetArrival(arrival time.Time) *tripScheduleBuilder {
	b.arrival = arrival
	return b
}

func (b *tripScheduleBuilder) Build() (TripScheduleModel, error) {
	return TripScheduleModel{
		tripId:         b.tripId,
		routeId:        b.routeId,
		boardingOpen:   b.boardingOpen,
		boardingClosed: b.boardingClosed,
		departure:      b.departure,
		arrival:        b.arrival,
	}, nil
}

func (b *tripScheduleBuilder) MustBuild() TripScheduleModel {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
