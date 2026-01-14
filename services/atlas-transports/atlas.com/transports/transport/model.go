package transport

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
	observationMapId       _map.Id
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

// ObservationMapId returns the observation map ID
func (m Model) ObservationMapId() _map.Id {
	return m.observationMapId
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
	return NewBuilder(m.Name()).
		SetId(m.Id()).
		SetStartMapId(m.StartMapId()).
		SetStagingMapId(m.StagingMapId()).
		SetEnRouteMapIds(m.EnRouteMapIds()).
		SetDestinationMapId(m.DestinationMapId()).
		SetObservationMapId(m.ObservationMapId()).
		SetState(m.state).
		SetSchedule(m.schedule).
		SetBoardingWindowDuration(m.boardingWindowDuration).
		SetPreDepartureDuration(m.preDepartureDuration).
		SetTravelDuration(m.travelDuration).
		SetCycleInterval(m.cycleInterval)
}

func (m Model) UpdateState(now time.Time) (Model, bool, error) {
	newState := m.processStateChange(now)
	updated, err := m.Builder().SetState(newState).Build()
	if err != nil {
		return Model{}, false, err
	}
	return updated, m.State() != newState, nil
}

func (m Model) processStateChange(now time.Time) RouteState {
	// Find the next trip
	var nextTrip *TripScheduleModel
	var inTransitTrip *TripScheduleModel
	var futureTrip *TripScheduleModel
	var arrivedTrip *TripScheduleModel

	// Get the current time of day
	nowTimeOfDay := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)

	for i := range m.Schedule() {
		trip := m.schedule[i]
		if trip.RouteId() == m.Id() {
			// Extract time of day from trip times
			tripDepartureTimeOfDay := time.Date(0, 1, 1, trip.Departure().Hour(), trip.Departure().Minute(), trip.Departure().Second(), trip.Departure().Nanosecond(), time.UTC)
			tripArrivalTimeOfDay := time.Date(0, 1, 1, trip.Arrival().Hour(), trip.Arrival().Minute(), trip.Arrival().Second(), trip.Arrival().Nanosecond(), time.UTC)

			// Handle cases where times cross midnight
			if tripArrivalTimeOfDay.Before(tripDepartureTimeOfDay) {
				// If arrival is before departure in time of day, it means arrival is on the next day
				if nowTimeOfDay.After(tripDepartureTimeOfDay) || nowTimeOfDay.Before(tripArrivalTimeOfDay) {
					// Current time is either after departure or before arrival (crossing midnight)
					if inTransitTrip == nil || tripDepartureTimeOfDay.After(time.Date(0, 1, 1, inTransitTrip.Departure().Hour(), inTransitTrip.Departure().Minute(), inTransitTrip.Departure().Second(), inTransitTrip.Departure().Nanosecond(), time.UTC)) {
						inTransitTrip = &trip
					}
				}
			} else {
				// Normal case (no midnight crossing)
				if nowTimeOfDay.After(tripDepartureTimeOfDay) && nowTimeOfDay.Before(tripArrivalTimeOfDay) {
					if inTransitTrip == nil || tripDepartureTimeOfDay.After(time.Date(0, 1, 1, inTransitTrip.Departure().Hour(), inTransitTrip.Departure().Minute(), inTransitTrip.Departure().Second(), inTransitTrip.Departure().Nanosecond(), time.UTC)) {
						inTransitTrip = &trip
					}
				}
			}

			// Handle future trips
			if tripDepartureTimeOfDay.After(nowTimeOfDay) {
				if futureTrip == nil || tripDepartureTimeOfDay.Before(time.Date(0, 1, 1, futureTrip.Departure().Hour(), futureTrip.Departure().Minute(), futureTrip.Departure().Second(), futureTrip.Departure().Nanosecond(), time.UTC)) {
					futureTrip = &trip
				}
			}

			// Handle arrived trips
			if tripArrivalTimeOfDay.Before(nowTimeOfDay) {
				if arrivedTrip == nil || tripArrivalTimeOfDay.After(time.Date(0, 1, 1, arrivedTrip.Arrival().Hour(), arrivedTrip.Arrival().Minute(), arrivedTrip.Arrival().Second(), arrivedTrip.Arrival().Nanosecond(), time.UTC)) {
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

	// Extract time of day from next trip times for comparison
	nextTripBoardingOpenTimeOfDay := time.Date(0, 1, 1, nextTrip.BoardingOpen().Hour(), nextTrip.BoardingOpen().Minute(), nextTrip.BoardingOpen().Second(), nextTrip.BoardingOpen().Nanosecond(), time.UTC)
	nextTripBoardingClosedTimeOfDay := time.Date(0, 1, 1, nextTrip.BoardingClosed().Hour(), nextTrip.BoardingClosed().Minute(), nextTrip.BoardingClosed().Second(), nextTrip.BoardingClosed().Nanosecond(), time.UTC)
	nextTripDepartureTimeOfDay := time.Date(0, 1, 1, nextTrip.Departure().Hour(), nextTrip.Departure().Minute(), nextTrip.Departure().Second(), nextTrip.Departure().Nanosecond(), time.UTC)
	nextTripArrivalTimeOfDay := time.Date(0, 1, 1, nextTrip.Arrival().Hour(), nextTrip.Arrival().Minute(), nextTrip.Arrival().Second(), nextTrip.Arrival().Nanosecond(), time.UTC)

	// Handle cases where times cross midnight
	if nextTripArrivalTimeOfDay.Before(nextTripDepartureTimeOfDay) {
		// If arrival is before departure in time of day, it means arrival is on the next day
		if nowTimeOfDay.Before(nextTripBoardingOpenTimeOfDay) && nowTimeOfDay.After(nextTripArrivalTimeOfDay) {
			return AwaitingReturn
		} else if nowTimeOfDay.Before(nextTripBoardingClosedTimeOfDay) {
			return OpenEntry
		} else if nowTimeOfDay.Before(nextTripDepartureTimeOfDay) {
			return LockedEntry
		} else {
			return InTransit
		}
	} else {
		// Normal case (no midnight crossing)
		if nowTimeOfDay.Before(nextTripBoardingOpenTimeOfDay) {
			return AwaitingReturn
		} else if nowTimeOfDay.Before(nextTripBoardingClosedTimeOfDay) {
			return OpenEntry
		} else if nowTimeOfDay.Before(nextTripDepartureTimeOfDay) {
			return LockedEntry
		} else if nowTimeOfDay.Before(nextTripArrivalTimeOfDay) {
			return InTransit
		} else if futureTrip != nil {
			return AwaitingReturn
		} else if arrivedTrip != nil {
			return AwaitingReturn
		} else {
			return OutOfService
		}
	}
}

func (m Model) State() RouteState {
	return m.state
}

func (m Model) Schedule() []TripScheduleModel {
	return m.schedule
}

// SharedVesselModel is the domain model for a shared vessel
type SharedVesselModel struct {
	id              uuid.UUID
	name            string
	routeAID        uuid.UUID
	routeBID        uuid.UUID
	turnaroundDelay time.Duration
}

// NewSharedVesselModel creates a new shared vessel model
func NewSharedVesselModel(
	id uuid.UUID,
	name string,
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
func (m SharedVesselModel) Id() uuid.UUID {
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

func (m TripScheduleModel) Builder() *TripScheduleBuilder {
	return NewTripScheduleBuilder().
		SetTripId(m.tripId).
		SetRouteId(m.routeId).
		SetBoardingOpen(m.boardingOpen).
		SetBoardingClosed(m.boardingClosed).
		SetDeparture(m.departure).
		SetArrival(m.arrival)
}
