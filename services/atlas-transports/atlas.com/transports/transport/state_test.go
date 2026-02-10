package transport

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachine_UpdateState(t *testing.T) {
	// Setup a fixed reference time for testing
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a test route
	routeID := uuid.New()
	route, err := NewBuilder("Test Route").
		SetStartMapId(100).
		SetStagingMapId(101).
		SetEnRouteMapIds([]_map.Id{102}).
		SetDestinationMapId(103).
		SetObservationMapId(104).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)
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
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         AwaitingReturn,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "During boarding window",
			currentTime: now.Add(7 * time.Minute),
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         OpenEntry,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After boarding closes but before departure",
			currentTime: now.Add(11 * time.Minute),
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         LockedEntry,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After departure but before arrival",
			currentTime: now.Add(15 * time.Minute),
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         InTransit,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "After arrival",
			currentTime: now.Add(25 * time.Minute),
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         OutOfService,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
		{
			name:        "Multiple trips - selects next trip",
			currentTime: now,
			trips: func() []TripScheduleModel {
				trip1Model, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(30 * time.Minute)).
					SetBoardingClosed(now.Add(35 * time.Minute)).
					SetDeparture(now.Add(37 * time.Minute)).
					SetArrival(now.Add(47 * time.Minute)).
					Build()
				trip2Model, _ := NewTripScheduleBuilder().
					SetTripId(trip2).
					SetRouteId(routeID).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip1Model, trip2Model}
			}(),
			expectedState:         AwaitingReturn,
			expectedNextDeparture: true,
			expectedBoardingEnds:  true,
		},
		{
			name:        "Trip for different route is ignored",
			currentTime: now,
			trips: func() []TripScheduleModel {
				trip, _ := NewTripScheduleBuilder().
					SetTripId(trip1).
					SetRouteId(uuid.New()). // Different route ID
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(12 * time.Minute)).
					SetArrival(now.Add(22 * time.Minute)).
					Build()
				return []TripScheduleModel{trip}
			}(),
			expectedState:         OutOfService,
			expectedNextDeparture: false,
			expectedBoardingEnds:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoute, err := route.Builder().SetSchedule(tt.trips).Build()
			require.NoError(t, err)

			// Update state based on test case
			testRoute, changed, err := testRoute.UpdateState(tt.currentTime)
			require.NoError(t, err)

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
	route, err := NewBuilder("Test Route").
		SetStartMapId(0).
		SetStagingMapId(0).
		SetEnRouteMapIds([]_map.Id{0}).
		SetDestinationMapId(0).
		SetObservationMapId(0).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	// Initially, state should be out of service
	initialState := route.State()
	assert.Equal(t, OutOfService, initialState, "Initial state should be out of service")

	trip1 := uuid.New()

	// Use a fixed reference time to avoid midnight-crossing issues with time-of-day comparisons
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	trip, err := NewTripScheduleBuilder().
		SetTripId(trip1).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(5 * time.Minute)).
		SetBoardingClosed(now.Add(10 * time.Minute)).
		SetDeparture(now.Add(12 * time.Minute)).
		SetArrival(now.Add(22 * time.Minute)).
		Build()
	require.NoError(t, err)
	trips := []TripScheduleModel{trip}
	route, err = route.Builder().SetSchedule(trips).Build()
	require.NoError(t, err)

	route, changed, err := route.UpdateState(now.Add(5 * time.Minute))
	require.NoError(t, err)

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
	route, err := NewBuilder("Test Route").
		SetStartMapId(0).
		SetStagingMapId(0).
		SetEnRouteMapIds([]_map.Id{0}).
		SetDestinationMapId(0).
		SetObservationMapId(0).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)
	trip1 := uuid.New()
	trip2 := uuid.New()
	trip3 := uuid.New()

	// Create multiple trips with different departure times
	tripModel1, err := NewTripScheduleBuilder().
		SetTripId(trip3).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(60 * time.Minute)).
		SetBoardingClosed(now.Add(65 * time.Minute)).
		SetDeparture(now.Add(67 * time.Minute)).
		SetArrival(now.Add(77 * time.Minute)).
		Build()
	require.NoError(t, err)
	tripModel2, err := NewTripScheduleBuilder().
		SetTripId(trip1).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(5 * time.Minute)).
		SetBoardingClosed(now.Add(10 * time.Minute)).
		SetDeparture(now.Add(12 * time.Minute)).
		SetArrival(now.Add(22 * time.Minute)).
		Build()
	require.NoError(t, err)
	tripModel3, err := NewTripScheduleBuilder().
		SetTripId(trip2).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(30 * time.Minute)).
		SetBoardingClosed(now.Add(35 * time.Minute)).
		SetDeparture(now.Add(37 * time.Minute)).
		SetArrival(now.Add(47 * time.Minute)).
		Build()
	require.NoError(t, err)

	trips := []TripScheduleModel{tripModel1, tripModel2, tripModel3}
	route, err = route.Builder().SetSchedule(trips).Build()
	require.NoError(t, err)

	// Update state
	route, changed, err := route.UpdateState(now)
	require.NoError(t, err)

	// Should select trip1 as it's the next one
	assert.Equal(t, AwaitingReturn, route.State())

	// First update should show state changed
	assert.True(t, changed, "StateChanged should be true for first update")

	// Move time forward to after trip1 but before trip2
	route, changed, err = route.UpdateState(now.Add(25 * time.Minute))
	require.NoError(t, err)

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
	route, err := NewBuilder("Test Route").
		SetStartMapId(100).
		SetStagingMapId(101).
		SetEnRouteMapIds([]_map.Id{102}).
		SetDestinationMapId(103).
		SetObservationMapId(104).
		SetId(routeID).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	trip1 := uuid.New()

	// Create a trip
	trip, err := NewTripScheduleBuilder().
		SetTripId(trip1).
		SetRouteId(routeID).
		SetBoardingOpen(now.Add(5 * time.Minute)).
		SetBoardingClosed(now.Add(10 * time.Minute)).
		SetDeparture(now.Add(12 * time.Minute)).
		SetArrival(now.Add(22 * time.Minute)).
		Build()
	require.NoError(t, err)

	trips := []TripScheduleModel{trip}
	route, err = route.Builder().SetSchedule(trips).Build()
	require.NoError(t, err)

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
			var updateErr error
			route, changed, updateErr = route.UpdateState(tc.currentTime)
			require.NoError(t, updateErr)

			assert.Equal(t, tc.expectedStatus, route.State(), "Status should match expected")
			assert.Equal(t, tc.stateChanged, changed, "StateChanged should match expected")
		})
	}
}
