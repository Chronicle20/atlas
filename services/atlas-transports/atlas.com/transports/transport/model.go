package transport

import (
	"time"

	"github.com/google/uuid"
)

// Model is the domain model for a transport route
type Model struct {
	id                     uuid.UUID
	name                   string
	startMapID             uint32
	stagingMapID           uint32
	enRouteMapID           uint32
	destinationMapID       uint32
	boardingWindowDuration time.Duration
	preDepartureDuration   time.Duration
	travelDuration         time.Duration
	cycleInterval          time.Duration
}

// NewModel creates a new transport route model
func NewModel(
	id uuid.UUID,
	name string,
	startMapID uint32,
	stagingMapID uint32,
	enRouteMapID uint32,
	destinationMapID uint32,
	boardingWindowDuration time.Duration,
	preDepartureDuration time.Duration,
	travelDuration time.Duration,
	cycleInterval time.Duration,
) Model {
	return Model{
		id:                     id,
		name:                   name,
		startMapID:             startMapID,
		stagingMapID:           stagingMapID,
		enRouteMapID:           enRouteMapID,
		destinationMapID:       destinationMapID,
		boardingWindowDuration: boardingWindowDuration,
		preDepartureDuration:   preDepartureDuration,
		travelDuration:         travelDuration,
		cycleInterval:          cycleInterval,
	}
}

// Id returns the route ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// Name returns the route name
func (m Model) Name() string {
	return m.name
}

// StartMapID returns the starting map ID
func (m Model) StartMapID() uint32 {
	return m.startMapID
}

// StagingMapID returns the staging map ID
func (m Model) StagingMapID() uint32 {
	return m.stagingMapID
}

// EnRouteMapID returns the en-route map ID
func (m Model) EnRouteMapID() uint32 {
	return m.enRouteMapID
}

// DestinationMapID returns the destination map ID
func (m Model) DestinationMapID() uint32 {
	return m.destinationMapID
}

// BoardingWindowDuration returns the boarding window duration
func (m Model) BoardingWindowDuration() time.Duration {
	return m.boardingWindowDuration
}

// PreDepartureDuration returns the pre-departure duration
func (m Model) PreDepartureDuration() time.Duration {
	return m.preDepartureDuration
}

// TravelDuration returns the travel duration
func (m Model) TravelDuration() time.Duration {
	return m.travelDuration
}

// CycleInterval returns the cycle interval
func (m Model) CycleInterval() time.Duration {
	return m.cycleInterval
}

// Builder is a builder for Model
type Builder struct {
	id                     uuid.UUID
	name                   string
	startMapID             uint32
	stagingMapID           uint32
	enRouteMapID           uint32
	destinationMapID       uint32
	boardingWindowDuration time.Duration
	preDepartureDuration   time.Duration
	travelDuration         time.Duration
	cycleInterval          time.Duration
}

// NewBuilder creates a new builder for Model
func NewBuilder() *Builder {
	return &Builder{
		id: uuid.New(),
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

// SetStartMapID sets the starting map ID
func (b *Builder) SetStartMapID(startMapID uint32) *Builder {
	b.startMapID = startMapID
	return b
}

// SetStagingMapID sets the staging map ID
func (b *Builder) SetStagingMapID(stagingMapID uint32) *Builder {
	b.stagingMapID = stagingMapID
	return b
}

// SetEnRouteMapID sets the en-route map ID
func (b *Builder) SetEnRouteMapID(enRouteMapID uint32) *Builder {
	b.enRouteMapID = enRouteMapID
	return b
}

// SetDestinationMapID sets the destination map ID
func (b *Builder) SetDestinationMapID(destinationMapID uint32) *Builder {
	b.destinationMapID = destinationMapID
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

// Build builds the Model
func (b *Builder) Build() Model {
	return NewModel(
		b.id,
		b.name,
		b.startMapID,
		b.stagingMapID,
		b.enRouteMapID,
		b.destinationMapID,
		b.boardingWindowDuration,
		b.preDepartureDuration,
		b.travelDuration,
		b.cycleInterval,
	)
}

// SharedVesselModel is the domain model for a shared vessel
type SharedVesselModel struct {
	id              string
	routeAID        uuid.UUID
	routeBID        uuid.UUID
	turnaroundDelay time.Duration
}

// NewSharedVesselModel creates a new shared vessel model
func NewSharedVesselModel(
	id string,
	routeAID uuid.UUID,
	routeBID uuid.UUID,
	turnaroundDelay time.Duration,
) SharedVesselModel {
	return SharedVesselModel{
		id:              id,
		routeAID:        routeAID,
		routeBID:        routeBID,
		turnaroundDelay: turnaroundDelay,
	}
}

// Id returns the shared vessel ID
func (m SharedVesselModel) Id() string {
	return m.id
}

// RouteAID returns the ID of route A
func (m SharedVesselModel) RouteAID() uuid.UUID {
	return m.routeAID
}

// RouteBID returns the ID of route B
func (m SharedVesselModel) RouteBID() uuid.UUID {
	return m.routeBID
}

// TurnaroundDelay returns the turnaround delay
func (m SharedVesselModel) TurnaroundDelay() time.Duration {
	return m.turnaroundDelay
}

// SharedVesselBuilder is a builder for SharedVesselModel
type SharedVesselBuilder struct {
	id              string
	routeAID        uuid.UUID
	routeBID        uuid.UUID
	turnaroundDelay time.Duration
}

// NewSharedVesselBuilder creates a new builder for SharedVesselModel
func NewSharedVesselBuilder() *SharedVesselBuilder {
	return &SharedVesselBuilder{
		id: uuid.New().String(),
	}
}

// SetId sets the shared vessel ID
func (b *SharedVesselBuilder) SetId(id string) *SharedVesselBuilder {
	b.id = id
	return b
}

// SetRouteAID sets the ID of route A
func (b *SharedVesselBuilder) SetRouteAID(routeAID uuid.UUID) *SharedVesselBuilder {
	b.routeAID = routeAID
	return b
}

// SetRouteBID sets the ID of route B
func (b *SharedVesselBuilder) SetRouteBID(routeBID uuid.UUID) *SharedVesselBuilder {
	b.routeBID = routeBID
	return b
}

// SetTurnaroundDelay sets the turnaround delay
func (b *SharedVesselBuilder) SetTurnaroundDelay(turnaroundDelay time.Duration) *SharedVesselBuilder {
	b.turnaroundDelay = turnaroundDelay
	return b
}

// Build builds the SharedVesselModel
func (b *SharedVesselBuilder) Build() SharedVesselModel {
	return NewSharedVesselModel(
		b.id,
		b.routeAID,
		b.routeBID,
		b.turnaroundDelay,
	)
}

// TripScheduleModel is the domain model for a trip schedule
type TripScheduleModel struct {
	tripID         string
	routeID        uuid.UUID
	boardingOpen   time.Time
	boardingClosed time.Time
	departure      time.Time
	arrival        time.Time
}

// NewTripScheduleModel creates a new trip schedule model
func NewTripScheduleModel(
	tripID string,
	routeID uuid.UUID,
	boardingOpen time.Time,
	boardingClosed time.Time,
	departure time.Time,
	arrival time.Time,
) TripScheduleModel {
	return TripScheduleModel{
		tripID:         tripID,
		routeID:        routeID,
		boardingOpen:   boardingOpen,
		boardingClosed: boardingClosed,
		departure:      departure,
		arrival:        arrival,
	}
}

// TripID returns the trip ID
func (m TripScheduleModel) TripID() string {
	return m.tripID
}

// RouteID returns the route ID
func (m TripScheduleModel) RouteID() uuid.UUID {
	return m.routeID
}

// BoardingOpen returns the boarding open time
func (m TripScheduleModel) BoardingOpen() time.Time {
	return m.boardingOpen
}

// BoardingClosed returns the boarding closed time
func (m TripScheduleModel) BoardingClosed() time.Time {
	return m.boardingClosed
}

// Departure returns the departure time
func (m TripScheduleModel) Departure() time.Time {
	return m.departure
}

// Arrival returns the arrival time
func (m TripScheduleModel) Arrival() time.Time {
	return m.arrival
}

// TripScheduleBuilder is a builder for TripScheduleModel
type TripScheduleBuilder struct {
	tripID         string
	routeID        uuid.UUID
	boardingOpen   time.Time
	boardingClosed time.Time
	departure      time.Time
	arrival        time.Time
}

// NewTripScheduleBuilder creates a new builder for TripScheduleModel
func NewTripScheduleBuilder() *TripScheduleBuilder {
	return &TripScheduleBuilder{}
}

// SetTripID sets the trip ID
func (b *TripScheduleBuilder) SetTripID(tripID string) *TripScheduleBuilder {
	b.tripID = tripID
	return b
}

// SetRouteID sets the route ID
func (b *TripScheduleBuilder) SetRouteID(routeID uuid.UUID) *TripScheduleBuilder {
	b.routeID = routeID
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

// Build builds the TripScheduleModel
func (b *TripScheduleBuilder) Build() TripScheduleModel {
	return NewTripScheduleModel(
		b.tripID,
		b.routeID,
		b.boardingOpen,
		b.boardingClosed,
		b.departure,
		b.arrival,
	)
}
