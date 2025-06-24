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
	route := NewBuilder("Test Route", 100, 101, 102, 103).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	trip1 := uuid.New()
	trip2 := uuid.New()

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
			expectedState:         OutOfService,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
		{
			name:        "Before boarding opens",
			currentTime: now,
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
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
					SetTripId(trip1).
					SetRouteId(routeID).
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
					SetTripId(trip1).
					SetRouteId(routeID).
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
					SetTripId(trip1).
					SetRouteId(routeID).
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
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         OutOfService,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
		{
			name:        "Multiple trips - selects next trip",
			currentTime: now,
			trips: []TripScheduleModel{
				NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(30 * time.Minute)).
					SetBoardingClosed(now.Add(35 * time.Minute)).
					SetDeparture(now.Add(37 * time.Minute)).
					SetArrival(now.Add(47 * time.Minute)).
					Build(),
				NewTripScheduleBuilder().
					SetTripId(trip2).
					SetRouteId(routeID).
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
					SetTripId(trip1).
					SetRouteId(uuid.New()). // Different route ID
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build(),
			},
			expectedState:         OutOfService,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoute := route.Builder().SetSchedule(tt.trips).Build()

			// Update state based on test case
			testRoute, changed := testRoute.UpdateState(tt.currentTime)

			// Assert state
			assert.Equal(t, tt.expectedState, testRoute.State(), "State should match expected")

			// For the first test, stateChanged should be false since there's no previous state
			if tt.name == "No trips scheduled" {
				assert.False(t, changed, "StateChanged should be false for first update")
			}
		})
	}
}

func TestStateMachine_GetState(t *testing.T) {
	// Create a test route
	routeID := uuid.New()
	route := NewBuilder("Test Route", 0, 0, 0, 0).
		SetId(routeID).
		Build()

	// Initially, state should be out of service
	initialState := route.State()
	assert.Equal(t, OutOfService, initialState, "Initial state should be out of service")

	trip1 := uuid.New()

	// Update state
	now := time.Now()
	trips := []TripScheduleModel{
		NewTripScheduleBuilder().
			SetTripId(trip1).
			SetRouteId(routeID).
			SetBoardingOpen(now.Add(5 * time.Minute)).
			SetBoardingClosed(now.Add(10 * time.Minute)).
			SetDeparture(now.Add(12 * time.Minute)).
			SetArrival(now.Add(22 * time.Minute)).
			Build(),
	}
	route = route.Builder().SetSchedule(trips).Build()

	route, changed := route.UpdateState(now.Add(5 * time.Minute))

	// GetState should return the updated state
	assert.Equal(t, OpenEntry, route.State(), "GetState should return the updated state")

	// First update should show state changed
	assert.True(t, changed, "StateChanged should be true for first update")
}

func TestStateMachine_MultipleTrips(t *testing.T) {
	// Setup a fixed reference time for testing
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a test route
	routeID := uuid.New()
	route := NewBuilder("Test Route", 0, 0, 0, 0).
		SetId(routeID).
		Build()
	trip1 := uuid.New()
	trip2 := uuid.New()
	trip3 := uuid.New()

	// Create multiple trips with different departure times
	trips := []TripScheduleModel{
		NewTripScheduleBuilder().
			SetTripId(trip3).
			SetRouteId(routeID).
			SetBoardingOpen(now.Add(60 * time.Minute)).
			SetBoardingClosed(now.Add(65 * time.Minute)).
			SetDeparture(now.Add(67 * time.Minute)).
			SetArrival(now.Add(77 * time.Minute)).
			Build(),
		NewTripScheduleBuilder().
			SetTripId(trip1).
			SetRouteId(routeID).
			SetBoardingOpen(now.Add(5 * time.Minute)).
			SetBoardingClosed(now.Add(10 * time.Minute)).
			SetDeparture(now.Add(12 * time.Minute)).
			SetArrival(now.Add(22 * time.Minute)).
			Build(),
		NewTripScheduleBuilder().
			SetTripId(trip2).
			SetRouteId(routeID).
			SetBoardingOpen(now.Add(30 * time.Minute)).
			SetBoardingClosed(now.Add(35 * time.Minute)).
			SetDeparture(now.Add(37 * time.Minute)).
			SetArrival(now.Add(47 * time.Minute)).
			Build(),
	}
	route = route.Builder().SetSchedule(trips).Build()

	// Update state
	route, changed := route.UpdateState(now)

	// Should select trip1 as it's the next one
	assert.Equal(t, AwaitingReturn, route.State())

	// First update should show state changed
	assert.True(t, changed, "StateChanged should be true for first update")

	// Move time forward to after trip1 but before trip2
	route, changed = route.UpdateState(now.Add(25 * time.Minute))

	// Should select trip2 as it's the next one
	assert.Equal(t, AwaitingReturn, route.State())

	// Status didn't change (still AwaitingReturn), but the trip did
	assert.False(t, changed, "StateChanged should be false when status doesn't change")
}

func TestStateMachine_StateChanged(t *testing.T) {
	// Setup a fixed reference time for testing
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a test route
	routeID := uuid.New()
	route := NewBuilder("Test Route", 100, 101, 102, 103).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	trip1 := uuid.New()

	// Create a trip
	trip := NewTripScheduleBuilder().
		SetTripId(trip1).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(5 * time.Minute)).
		SetBoardingClosed(now.Add(10 * time.Minute)).
		SetDeparture(now.Add(12 * time.Minute)).
		SetArrival(now.Add(22 * time.Minute)).
		Build()

	trips := []TripScheduleModel{trip}
	route = route.Builder().SetSchedule(trips).Build()

	// Test cases for state changes
	testCases := []struct {
		name           string
		currentTime    time.Time
		expectedStatus RouteState
		stateChanged   bool
	}{
		{
			name:           "Initial state",
			currentTime:    now,
			expectedStatus: AwaitingReturn,
			stateChanged:   true, // First update always changes state
		},
		{
			name:           "Same state (AwaitingReturn)",
			currentTime:    now.Add(1 * time.Minute),
			expectedStatus: AwaitingReturn,
			stateChanged:   false, // Status didn't change
		},
		{
			name:           "Change to OpenEntry",
			currentTime:    now.Add(6 * time.Minute), // During boarding window
			expectedStatus: OpenEntry,
			stateChanged:   true, // Status changed from AwaitingReturn to OpenEntry
		},
		{
			name:           "Same state (OpenEntry)",
			currentTime:    now.Add(7 * time.Minute), // Still during boarding window
			expectedStatus: OpenEntry,
			stateChanged:   false, // Status didn't change
		},
		{
			name:           "Change to LockedEntry",
			currentTime:    now.Add(11 * time.Minute), // After boarding closes but before departure
			expectedStatus: LockedEntry,
			stateChanged:   true, // Status changed from OpenEntry to LockedEntry
		},
		{
			name:           "Change to InTransit",
			currentTime:    now.Add(15 * time.Minute), // After departure but before arrival
			expectedStatus: InTransit,
			stateChanged:   true, // Status changed from LockedEntry to InTransit
		},
		{
			name:           "Change back to AwaitingReturn",
			currentTime:    now.Add(25 * time.Minute), // After arrival
			expectedStatus: OutOfService,
			stateChanged:   true, // Status changed from InTransit to AwaitingReturn
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var changed bool
			route, changed = route.UpdateState(tc.currentTime)

			assert.Equal(t, tc.expectedStatus, route.State(), "Status should match expected")
			assert.Equal(t, tc.stateChanged, changed, "StateChanged should match expected")
		})
	}
}
