package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine_UpdateState(t *testing.T) {
	// Setup a fixed reference time for testing
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a test route
	routeID := uuid.New()
	route := NewBuilder().
		SetId(routeID).
		SetName("Test Route").
		SetStartMapID(100).
		SetStagingMapID(101).
		SetEnRouteMapID(102).
		SetDestinationMapID(103).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	// Create a state machine for the route
	sm := NewStateMachine(route)

	// Test cases
	tests := []struct {
		name                  string
		currentTime           time.Time
		trips                 []TripScheduleModel
		expectedState         RouteState
		expectedNextDeparture bool
		expectedBoardingEnds  bool
	}{
		{
			name:                  "No trips scheduled",
			currentTime:           now,
			trips:                 []TripScheduleModel{},
			expectedState:         AwaitingReturn,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
		{
			name:        "Before boarding opens",
			currentTime: now,
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         AwaitingReturn,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "During boarding window",
			currentTime: now.Add(7 * time.Minute),
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         OpenEntry,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After boarding closes but before departure",
			currentTime: now.Add(11 * time.Minute),
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         LockedEntry,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After departure but before arrival",
			currentTime: now.Add(15 * time.Minute),
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         InTransit,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After arrival",
			currentTime: now.Add(25 * time.Minute),
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         AwaitingReturn,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
		{
			name:        "Multiple trips - selects next trip",
			currentTime: now,
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(30 * time.Minute)).
					SetBoardingClosed(now.Add(35 * time.Minute)).
					SetDeparture(now.Add(37 * time.Minute)).
					SetArrival(now.Add(47 * time.Minute)).
					Build(),
				NewTripScheduleBuilder().
					SetTripID("trip2").
					SetRouteID(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         AwaitingReturn,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "Trip for different route is ignored",
			currentTime: now,
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripID("trip1").
					SetRouteID(uuid.New()). // Different route ID
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         AwaitingReturn,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update state based on test case
			state := sm.UpdateState(tt.currentTime, tt.trips)

			// Assert state
			assert.Equal(t, tt.expectedState, state.Status(), "State should match expected")

			// Assert next departure and boarding ends times
			if tt.expectedNextDeparture {
				assert.False(t, state.NextDeparture().IsZero(), "NextDeparture should be set")
			} else {
				assert.True(t, state.NextDeparture().IsZero(), "NextDeparture should not be set")
			}

			if tt.expectedBoardingEnds {
				assert.False(t, state.BoardingEnds().IsZero(), "BoardingEnds should be set")
			} else {
				assert.True(t, state.BoardingEnds().IsZero(), "BoardingEnds should not be set")
			}
		})
	}
}

func TestStateMachine_GetState(t *testing.T) {
	// Create a test route
	routeID := uuid.New()
	route := NewBuilder().
		SetId(routeID).
		SetName("Test Route").
		Build()

	// Create a state machine for the route
	sm := NewStateMachine(route)

	// Initially, state should be empty
	initialState := sm.GetState()
	assert.Equal(t, RouteStateModel{}, initialState, "Initial state should be empty")

	// Update state
	now := time.Now()
	trips := []TripScheduleModel{
		NewTripScheduleBuilder().
			SetTripID("trip1").
			SetRouteID(routeID).
			SetBoardingOpen(now.Add(5 * time.Minute)).
			SetBoardingClosed(now.Add(10 * time.Minute)).
			SetDeparture(now.Add(12 * time.Minute)).
			SetArrival(now.Add(22 * time.Minute)).
			Build(),
	}

	updatedState := sm.UpdateState(now, trips)

	// GetState should return the updated state
	retrievedState := sm.GetState()
	assert.Equal(t, updatedState, retrievedState, "GetState should return the updated state")
}

func TestStateMachine_MultipleTrips(t *testing.T) {
	// Setup a fixed reference time for testing
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a test route
	routeID := uuid.New()
	route := NewBuilder().
		SetId(routeID).
		SetName("Test Route").
		Build()

	// Create a state machine for the route
	sm := NewStateMachine(route)

	// Create multiple trips with different departure times
	trips := []TripScheduleModel{
		NewTripScheduleBuilder().
			SetTripID("trip3").
			SetRouteID(routeID).
			SetBoardingOpen(now.Add(60 * time.Minute)).
			SetBoardingClosed(now.Add(65 * time.Minute)).
			SetDeparture(now.Add(67 * time.Minute)).
			SetArrival(now.Add(77 * time.Minute)).
			Build(),
		NewTripScheduleBuilder().
			SetTripID("trip1").
			SetRouteID(routeID).
			SetBoardingOpen(now.Add(5 * time.Minute)).
			SetBoardingClosed(now.Add(10 * time.Minute)).
			SetDeparture(now.Add(12 * time.Minute)).
			SetArrival(now.Add(22 * time.Minute)).
			Build(),
		NewTripScheduleBuilder().
			SetTripID("trip2").
			SetRouteID(routeID).
			SetBoardingOpen(now.Add(30 * time.Minute)).
			SetBoardingClosed(now.Add(35 * time.Minute)).
			SetDeparture(now.Add(37 * time.Minute)).
			SetArrival(now.Add(47 * time.Minute)).
			Build(),
	}

	// Update state
	state := sm.UpdateState(now, trips)

	// Should select trip1 as it's the next one
	assert.Equal(t, AwaitingReturn, state.Status())
	assert.Equal(t, trips[1].Departure(), state.NextDeparture())
	assert.Equal(t, trips[1].BoardingClosed(), state.BoardingEnds())

	// Move time forward to after trip1 but before trip2
	state = sm.UpdateState(now.Add(25*time.Minute), trips)

	// Should select trip2 as it's the next one
	assert.Equal(t, AwaitingReturn, state.Status())
	assert.Equal(t, trips[2].Departure(), state.NextDeparture())
	assert.Equal(t, trips[2].BoardingClosed(), state.BoardingEnds())
}
