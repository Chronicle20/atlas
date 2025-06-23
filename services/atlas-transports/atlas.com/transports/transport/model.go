package transport

import (
	"time"

	"github.com/google/uuid"
)

// Model is the domain model for a transport route
type Model struct {
	id                     uuid.UUID
	name                   string
	startMapId             uint32
	stagingMapId           uint32
	enRouteMapId           uint32
	destinationMapId       uint32
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
func (m Model) StartMapId() uint32 {
	return m.startMapId
}

// StagingMapId returns the staging map ID
func (m Model) StagingMapId() uint32 {
	return m.stagingMapId
}

// EnRouteMapId returns the en-route map ID
func (m Model) EnRouteMapId() uint32 {
	return m.enRouteMapId
}

// DestinationMapId returns the destination map ID
func (m Model) DestinationMapId() uint32 {
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

func (m Model) Builder() *Builder {
	return NewBuilder(m.Name(), m.StartMapId(), m.StagingMapId(), m.EnRouteMapId(), m.DestinationMapId()).
		SetId(m.Id()).
		SetState(m.state).
		SetSchedule(m.schedule).
		SetBoardingWindowDuration(m.boardingWindowDuration).
		SetPreDepartureDuration(m.preDepartureDuration).
		SetTravelDuration(m.travelDuration).
		SetCycleInterval(m.cycleInterval)
}

func (m Model) UpdateState(now time.Time) (Model, bool) {
	newState := m.processStateChange(now)
	return m.Builder().SetState(newState).Build(), m.State() != newState
}

func (m Model) processStateChange(now time.Time) RouteState {
	// Find the next trip
	var nextTrip *TripScheduleModel
	var inTransitTrip *TripScheduleModel
	var futureTrip *TripScheduleModel
	var arrivedTrip *TripScheduleModel

	for i := range m.Schedule() {
		trip := m.schedule[i]
		if trip.RouteId() == m.Id() {
			// Check if the trip is currently in transit (departed but not arrived)
			if trip.Departure().Before(now) && trip.Arrival().After(now) {
				if inTransitTrip == nil || trip.Departure().After(inTransitTrip.Departure()) {
					// For in-transit trips, prefer the most recently departed one
					inTransitTrip = &trip
				}
			} else if trip.Departure().After(now) {
				// For future trips, prefer the one departing soonest
				if futureTrip == nil || trip.Departure().Before(futureTrip.Departure()) {
					futureTrip = &trip
				}
			} else if trip.Arrival().Before(now) {
				// For arrived trips, prefer the most recently arrived one
				if arrivedTrip == nil || trip.Arrival().After(arrivedTrip.Arrival()) {
					arrivedTrip = &trip
				}
			}
		}
	}

	// Prioritize in-transit trips over future trips
	if inTransitTrip != nil {
		nextTrip = inTransitTrip
	} else {
		nextTrip = futureTrip
	}

	// If no next trip, set state to awaiting_return
	if nextTrip == nil {
		return OutOfService
	}

	// Determine the state based on the current time and next trip
	if now.Before(nextTrip.BoardingOpen()) {
		return AwaitingReturn
	} else if now.Before(nextTrip.BoardingClosed()) {
		return OpenEntry
	} else if now.Before(nextTrip.Departure()) {
		return LockedEntry
	} else if now.Before(nextTrip.Arrival()) {
		return InTransit
	} else if futureTrip != nil {
		return AwaitingReturn
	} else if arrivedTrip != nil {
		return AwaitingReturn
	} else {
		return OutOfService
	}
}

func (m Model) State() RouteState {
	return m.state
}

func (m Model) Schedule() []TripScheduleModel {
	return m.schedule
}

// Builder is a builder for Model
type Builder struct {
	id                     uuid.UUID
	name                   string
	startMapId             uint32
	stagingMapId           uint32
	enRouteMapId           uint32
	destinationMapId       uint32
	state                  RouteState
	schedule               []TripScheduleModel
	boardingWindowDuration time.Duration
	preDepartureDuration   time.Duration
	travelDuration         time.Duration
	cycleInterval          time.Duration
}

// NewBuilder creates a new builder for Model
func NewBuilder(name string, startMapId uint32, stagingMapId uint32, enRouteMapId uint32, destinationMapId uint32) *Builder {
	return &Builder{
		id:               uuid.New(),
		name:             name,
		startMapId:       startMapId,
		stagingMapId:     stagingMapId,
		enRouteMapId:     enRouteMapId,
		destinationMapId: destinationMapId,
		state:            OutOfService,
		schedule:         []TripScheduleModel{},
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
func (b *Builder) SetStartMapId(startMapId uint32) *Builder {
	b.startMapId = startMapId
	return b
}

// SetStagingMapId sets the staging map ID
func (b *Builder) SetStagingMapId(stagingMapId uint32) *Builder {
	b.stagingMapId = stagingMapId
	return b
}

// SetEnRouteMapId sets the en-route map ID
func (b *Builder) SetEnRouteMapId(enRouteMapId uint32) *Builder {
	b.enRouteMapId = enRouteMapId
	return b
}

// SetDestinationMapId sets the destination map ID
func (b *Builder) SetDestinationMapId(destinationMapId uint32) *Builder {
	b.destinationMapId = destinationMapId
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
	return Model{
		id:                     b.id,
		name:                   b.name,
		startMapId:             b.startMapId,
		stagingMapId:           b.stagingMapId,
		enRouteMapId:           b.enRouteMapId,
		destinationMapId:       b.destinationMapId,
		state:                  b.state,
		schedule:               b.schedule,
		boardingWindowDuration: b.boardingWindowDuration,
		preDepartureDuration:   b.preDepartureDuration,
		travelDuration:         b.travelDuration,
		cycleInterval:          b.cycleInterval,
	}
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

func (m TripScheduleModel) Builder() *TripScheduleBuilder {
	return NewTripScheduleBuilder().
		SetTripId(m.tripId).
		SetRouteId(m.routeId).
		SetBoardingOpen(m.boardingOpen).
		SetBoardingClosed(m.boardingClosed).
		SetDeparture(m.departure).
		SetArrival(m.arrival)
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

// Build builds the TripScheduleModel
func (b *TripScheduleBuilder) Build() TripScheduleModel {
	return NewTripScheduleModel(
		b.tripId,
		b.routeId,
		b.boardingOpen,
		b.boardingClosed,
		b.departure,
		b.arrival,
	)
}
