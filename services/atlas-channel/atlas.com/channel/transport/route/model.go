package route

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"time"

	"github.com/google/uuid"
)

// Model is the domain model for a transport route
type Model struct {
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

// Id returns the route ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// Name returns the route name
func (m Model) Name() string {
	return m.name
}

// StartMapId returns the starting map ID
func (m Model) StartMapId() _map.Id {
	return m.startMapId
}

// StagingMapId returns the staging map ID
func (m Model) StagingMapId() _map.Id {
	return m.stagingMapId
}

// EnRouteMapIds returns the en-route map IDs
func (m Model) EnRouteMapIds() []_map.Id {
	return m.enRouteMapIds
}

// DestinationMapId returns the destination map ID
func (m Model) DestinationMapId() _map.Id {
	return m.destinationMapId
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

func (m Model) State() RouteState {
	return m.state
}

func (m Model) Schedule() []TripScheduleModel {
	return m.schedule
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

// TripScheduleModel is the domain model for a trip schedule
type TripScheduleModel struct {
	tripId         uuid.UUID
	routeId        uuid.UUID
	boardingOpen   time.Time
	boardingClosed time.Time
	departure      time.Time
	arrival        time.Time
}

// NewTripScheduleModel creates a new trip schedule model
func NewTripScheduleModel(tripId uuid.UUID, routeId uuid.UUID, boardingOpen time.Time, boardingClosed time.Time, departure time.Time, arrival time.Time) TripScheduleModel {
	return TripScheduleModel{
		tripId:         tripId,
		routeId:        routeId,
		boardingOpen:   boardingOpen,
		boardingClosed: boardingClosed,
		departure:      departure,
		arrival:        arrival,
	}
}

// TripId returns the trip ID
func (m TripScheduleModel) TripId() uuid.UUID {
	return m.tripId
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

func (m TripScheduleModel) RouteId() uuid.UUID {
	return m.routeId
}
