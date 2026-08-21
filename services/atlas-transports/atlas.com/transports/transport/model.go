package transport

import (
	"time"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

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

// Transition is the result of evaluating a route's schedule against a moment
// in time: the state the route is in now, the state it moves to next, and the
// absolute instant of that move.
//
// Trip-schedule timestamps carry the date of the day the schedule was computed
// and only their time-of-day component is meaningful (the schedule is computed
// once per reconcile; the 1-second ticker only re-derives state from it). NextAt
// is that time-of-day boundary projected onto the first instant strictly after
// `now`, so - unlike a raw trip row - it is safe to render as an absolute
// timestamp. When State is OutOfService there is no boundary: NextState is ""
// and NextAt is the zero time.
type Transition struct {
	State     RouteState
	NextState RouteState
	NextAt    time.Time
	// TripId names the trip this transition is about, and DepartedAt is that
	// trip's departure time-of-day materialized onto the calendar day it
	// actually departed — the previous day when a midnight-crossing trip is
	// observed after midnight. Together with the route id and the tenant
	// they are the inputs to VoyageId (design §7.1). Both are zero when
	// State is OutOfService, where there is no selected trip.
	TripId     uuid.UUID
	DepartedAt time.Time
	// ArrivedTripId and ArrivedDepartedAt name the voyage that just ended -
	// the trip whose arrival is the most recent at or before now (aka
	// justArrivedTrip) - independent of TripId/DepartedAt, which name the
	// trip the transition's *state* is about. They are populated on every
	// return path of Evaluate regardless of State, so arrival side effects
	// can identify the arrived voyage even when the route lands somewhere
	// other than AwaitingReturn. Both are zero when no trip has arrived as
	// of now.
	ArrivedTripId     uuid.UUID
	ArrivedDepartedAt time.Time
}

// timeOfDay strips the date from t, leaving only a comparable time of day.
// Every schedule comparison in this file goes through it.
func timeOfDay(t time.Time) time.Time {
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// materializeBoundary projects a time-of-day boundary onto the first instant
// strictly after now, in now's own date and location - the same frame the
// time-of-day comparisons use, so the state and the boundary can never
// disagree about which side of a transition we are on.
func materializeBoundary(now time.Time, boundary time.Time) time.Time {
	at := time.Date(now.Year(), now.Month(), now.Day(),
		boundary.Hour(), boundary.Minute(), boundary.Second(), boundary.Nanosecond(),
		now.Location())
	if !at.After(now) {
		at = at.Add(24 * time.Hour)
	}
	return at
}

// materializeDeparture projects a trip's departure time-of-day onto the
// calendar day the trip actually departed, relative to now. For a same-day
// trip that is now's date. For a midnight-crossing trip observed after the
// crossing (now's time-of-day is before the departure time-of-day), the
// departure was yesterday — which is exactly the case that would otherwise
// make VOYAGE_ARRIVED derive a different voyage id than VOYAGE_DEPARTED did.
func materializeDeparture(now time.Time, departure time.Time) time.Time {
	at := time.Date(now.Year(), now.Month(), now.Day(),
		departure.Hour(), departure.Minute(), departure.Second(), departure.Nanosecond(),
		now.Location())
	if at.After(now) {
		at = at.Add(-24 * time.Hour)
	}
	return at
}

// Evaluate derives the route's state at `now` together with the transition it
// is counting down to. The trip-selection and branch structure is the state
// machine this service has always run; each branch now also names the boundary
// it is waiting on.
func (m Model) Evaluate(now time.Time) Transition {
	var nextTrip *TripScheduleModel
	var inTransitTrip *TripScheduleModel
	var futureTrip *TripScheduleModel
	// justArrivedTrip is, among this route's trips, the one whose arrival
	// time-of-day is the latest at or before nowTimeOfDay - the trip that has
	// just landed. It is what AwaitingReturn actually reports identity for
	// (see the AwaitingReturn branches below); nextTrip alone answers "what's
	// coming up next", which one tick past an arrival is a different trip.
	var justArrivedTrip *TripScheduleModel

	nowTimeOfDay := timeOfDay(now)

	for i := range m.Schedule() {
		trip := m.schedule[i]
		if trip.RouteId() != m.Id() {
			continue
		}

		tripDepartureTimeOfDay := timeOfDay(trip.Departure())
		tripArrivalTimeOfDay := timeOfDay(trip.Arrival())

		if tripArrivalTimeOfDay.Before(tripDepartureTimeOfDay) {
			// Arrival is before departure in time of day: the trip crosses midnight.
			if nowTimeOfDay.After(tripDepartureTimeOfDay) || nowTimeOfDay.Before(tripArrivalTimeOfDay) {
				if inTransitTrip == nil || tripDepartureTimeOfDay.After(timeOfDay(inTransitTrip.Departure())) {
					inTransitTrip = &trip
				}
			}
			// The trip's arrival sits on the low side of the zero-date axis
			// (e.g. 00:30) while its departure sits on the high side (e.g.
			// 23:30); "already arrived" is the region between them, the
			// complement of the in-transit wraparound above.
			if !nowTimeOfDay.Before(tripArrivalTimeOfDay) && nowTimeOfDay.Before(tripDepartureTimeOfDay) {
				if justArrivedTrip == nil || tripArrivalTimeOfDay.After(timeOfDay(justArrivedTrip.Arrival())) {
					justArrivedTrip = &trip
				}
			}
		} else {
			if nowTimeOfDay.After(tripDepartureTimeOfDay) && nowTimeOfDay.Before(tripArrivalTimeOfDay) {
				if inTransitTrip == nil || tripDepartureTimeOfDay.After(timeOfDay(inTransitTrip.Departure())) {
					inTransitTrip = &trip
				}
			}
			if !nowTimeOfDay.Before(tripArrivalTimeOfDay) {
				if justArrivedTrip == nil || tripArrivalTimeOfDay.After(timeOfDay(justArrivedTrip.Arrival())) {
					justArrivedTrip = &trip
				}
			}
		}

		if tripDepartureTimeOfDay.After(nowTimeOfDay) {
			if futureTrip == nil || tripDepartureTimeOfDay.Before(timeOfDay(futureTrip.Departure())) {
				futureTrip = &trip
			}
		}
	}

	// Prioritize in-transit trips over future trips
	if inTransitTrip != nil {
		nextTrip = inTransitTrip
	} else {
		nextTrip = futureTrip
	}

	// arrivedTripId and arrivedDepartedAt are justArrivedTrip's identity,
	// materialized the same way the selected trip's identity is. They are
	// zero when no trip has arrived as of now. Computed once here so every
	// return path below - including the nextTrip == nil early return, which
	// happens before nextTrip-based identity would otherwise be reachable -
	// can carry it.
	var arrivedTripId uuid.UUID
	var arrivedDepartedAt time.Time
	if justArrivedTrip != nil {
		arrivedTripId = justArrivedTrip.TripId()
		arrivedDepartedAt = materializeDeparture(now, timeOfDay(justArrivedTrip.Departure()))
	}

	if nextTrip == nil {
		return Transition{
			State:             OutOfService,
			ArrivedTripId:     arrivedTripId,
			ArrivedDepartedAt: arrivedDepartedAt,
		}
	}

	to := func(state RouteState, next RouteState, boundary time.Time) Transition {
		return Transition{
			State:             state,
			NextState:         next,
			NextAt:            materializeBoundary(now, boundary),
			TripId:            nextTrip.TripId(),
			DepartedAt:        materializeDeparture(now, timeOfDay(nextTrip.Departure())),
			ArrivedTripId:     arrivedTripId,
			ArrivedDepartedAt: arrivedDepartedAt,
		}
	}

	boardingOpen := timeOfDay(nextTrip.BoardingOpen())
	boardingClosed := timeOfDay(nextTrip.BoardingClosed())
	departure := timeOfDay(nextTrip.Departure())
	arrival := timeOfDay(nextTrip.Arrival())

	if arrival.Before(departure) {
		// Midnight-crossing trip. `arrival` sits on the low side of the
		// zero-date axis (e.g. 00:30) while boardingOpen/Closed/departure sit
		// on the high side (e.g. 22:30-23:30), so a post-midnight observation
		// (e.g. 00:10) numerically precedes boardingClosed/departure too - it
		// must be classified against `arrival` first, or it is misread as
		// pre-boarding on the wrong side of the crossing.
		if nowTimeOfDay.Before(arrival) {
			return to(InTransit, AwaitingReturn, arrival)
		} else if nowTimeOfDay.Before(boardingOpen) {
			return to(AwaitingReturn, OpenEntry, boardingOpen)
		} else if nowTimeOfDay.Before(boardingClosed) {
			return to(OpenEntry, LockedEntry, boardingClosed)
		} else if nowTimeOfDay.Before(departure) {
			return to(LockedEntry, InTransit, departure)
		}
		return to(InTransit, AwaitingReturn, arrival)
	}

	if nowTimeOfDay.Before(boardingOpen) {
		return to(AwaitingReturn, OpenEntry, boardingOpen)
	} else if nowTimeOfDay.Before(boardingClosed) {
		return to(OpenEntry, LockedEntry, boardingClosed)
	} else if nowTimeOfDay.Before(departure) {
		return to(LockedEntry, InTransit, departure)
	} else if nowTimeOfDay.Before(arrival) {
		return to(InTransit, AwaitingReturn, arrival)
	}
	return Transition{
		State:             OutOfService,
		ArrivedTripId:     arrivedTripId,
		ArrivedDepartedAt: arrivedDepartedAt,
	}
}

func (m Model) processStateChange(now time.Time) RouteState {
	return m.Evaluate(now).State
}

// UpdateStateWithTransition is UpdateState plus the Transition the new state
// was derived from. processStateChange already computes and discards everything
// but State; callers that need the trip identity (voyage events) read it here
// rather than calling Evaluate a second time with a slightly different `now`.
func (m Model) UpdateStateWithTransition(now time.Time) (Model, bool, Transition, error) {
	tr := m.Evaluate(now)
	updated, err := m.Builder().SetState(tr.State).Build()
	if err != nil {
		return Model{}, false, Transition{}, err
	}
	return updated, m.State() != tr.State, tr, nil
}

func (m Model) UpdateState(now time.Time) (Model, bool, error) {
	updated, changed, _, err := m.UpdateStateWithTransition(now)
	return updated, changed, err
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
	name            string
	routeAID        string
	routeBID        string
	turnaroundDelay time.Duration
}

// NewSharedVesselModel creates a new shared vessel model
func NewSharedVesselModel(
	id string,
	name string,
	routeAID string,
	routeBID string,
	turnaroundDelay time.Duration,
) SharedVesselModel {
	return SharedVesselModel{
		id:              id,
		name:            name,
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
func (m SharedVesselModel) RouteAID() string {
	return m.routeAID
}

// RouteBID returns the ID of route B
func (m SharedVesselModel) RouteBID() string {
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
