package transport

import (
	"github.com/google/uuid"
	"time"
)

// RouteState represents the state of a transport route
type RouteState string

const (
	// AwaitingReturn indicates that the vessel is not yet available
	AwaitingReturn RouteState = "awaiting_return"

	// OpenEntry indicates that players can board
	OpenEntry RouteState = "open_entry"

	// LockedEntry indicates that boarding is closed and the vessel is in pre-departure phase
	LockedEntry RouteState = "locked_entry"

	// InTransit indicates that characters are in the en-route map
	InTransit RouteState = "in_transit"
)

// RouteStateModel represents the current state of a transport route
type RouteStateModel struct {
	routeID       uuid.UUID
	status        RouteState
	nextDeparture time.Time
	boardingEnds  time.Time
}

// NewRouteStateModel creates a new route state model
func NewRouteStateModel(
	routeID uuid.UUID,
	status RouteState,
	nextDeparture time.Time,
	boardingEnds time.Time,
) RouteStateModel {
	return RouteStateModel{
		routeID:       routeID,
		status:        status,
		nextDeparture: nextDeparture,
		boardingEnds:  boardingEnds,
	}
}

// RouteID returns the route ID
func (m RouteStateModel) RouteID() uuid.UUID {
	return m.routeID
}

// Status returns the route status
func (m RouteStateModel) Status() RouteState {
	return m.status
}

// NextDeparture returns the next departure time
func (m RouteStateModel) NextDeparture() time.Time {
	return m.nextDeparture
}

// BoardingEnds returns the time when boarding ends
func (m RouteStateModel) BoardingEnds() time.Time {
	return m.boardingEnds
}

// RouteStateBuilder is a builder for RouteStateModel
type RouteStateBuilder struct {
	routeID       uuid.UUID
	status        RouteState
	nextDeparture time.Time
	boardingEnds  time.Time
}

// NewRouteStateBuilder creates a new builder for RouteStateModel
func NewRouteStateBuilder() *RouteStateBuilder {
	return &RouteStateBuilder{
		status: AwaitingReturn,
	}
}

// SetRouteID sets the route ID
func (b *RouteStateBuilder) SetRouteID(routeID uuid.UUID) *RouteStateBuilder {
	b.routeID = routeID
	return b
}

// SetStatus sets the route status
func (b *RouteStateBuilder) SetStatus(status RouteState) *RouteStateBuilder {
	b.status = status
	return b
}

// SetNextDeparture sets the next departure time
func (b *RouteStateBuilder) SetNextDeparture(nextDeparture time.Time) *RouteStateBuilder {
	b.nextDeparture = nextDeparture
	return b
}

// SetBoardingEnds sets the time when boarding ends
func (b *RouteStateBuilder) SetBoardingEnds(boardingEnds time.Time) *RouteStateBuilder {
	b.boardingEnds = boardingEnds
	return b
}

// Build builds the RouteStateModel
func (b *RouteStateBuilder) Build() RouteStateModel {
	return NewRouteStateModel(
		b.routeID,
		b.status,
		b.nextDeparture,
		b.boardingEnds,
	)
}

// StateMachine manages the state transitions for a transport route
type StateMachine struct {
	route Model
	state RouteStateModel
}

// NewStateMachine creates a new state machine for a transport route
func NewStateMachine(route Model) *StateMachine {
	return &StateMachine{
		route: route,
	}
}

// UpdateState updates the state of the route based on the current time and trip schedule
func (sm *StateMachine) UpdateState(now time.Time, trips []TripScheduleModel) RouteStateModel {
	// Find the next trip
	var nextTrip *TripScheduleModel
	var inTransitTrip *TripScheduleModel
	var futureTrip *TripScheduleModel
	var arrivedTrip *TripScheduleModel

	for i := range trips {
		trip := trips[i]
		if trip.RouteID() == sm.route.Id() {
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
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(AwaitingReturn).
			Build()
		return sm.state
	}

	// Determine the state based on the current time and next trip
	if now.Before(nextTrip.BoardingOpen()) {
		// Before boarding opens
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(AwaitingReturn).
			SetNextDeparture(nextTrip.Departure()).
			SetBoardingEnds(nextTrip.BoardingClosed()).
			Build()
	} else if now.Before(nextTrip.BoardingClosed()) {
		// During boarding window
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(OpenEntry).
			SetNextDeparture(nextTrip.Departure()).
			SetBoardingEnds(nextTrip.BoardingClosed()).
			Build()
	} else if now.Before(nextTrip.Departure()) {
		// After boarding closes but before departure
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(LockedEntry).
			SetNextDeparture(nextTrip.Departure()).
			SetBoardingEnds(nextTrip.BoardingClosed()).
			Build()
	} else if now.Before(nextTrip.Arrival()) {
		// After departure but before arrival
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(InTransit).
			SetNextDeparture(nextTrip.Departure()).
			SetBoardingEnds(nextTrip.BoardingClosed()).
			Build()
	} else if futureTrip != nil {
		// After arrival, but there's a future trip
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(AwaitingReturn).
			SetNextDeparture(futureTrip.Departure()).
			SetBoardingEnds(futureTrip.BoardingClosed()).
			Build()
	} else if arrivedTrip != nil {
		// After arrival, no future trips, but we have an arrived trip
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(AwaitingReturn).
			SetNextDeparture(arrivedTrip.Departure()).
			SetBoardingEnds(arrivedTrip.BoardingClosed()).
			Build()
	} else {
		// No trips at all
		sm.state = NewRouteStateBuilder().
			SetRouteID(sm.route.Id()).
			SetStatus(AwaitingReturn).
			Build()
	}

	return sm.state
}

// GetState returns the current state of the route
func (sm *StateMachine) GetState() RouteStateModel {
	return sm.state
}
