package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func schedulesPerRoute(trips []TripScheduleModel) map[uuid.UUID][]TripScheduleModel {
	result := make(map[uuid.UUID][]TripScheduleModel)
	for _, trip := range trips {
		if _, ok := result[trip.RouteId()]; !ok {
			result[trip.RouteId()] = make([]TripScheduleModel, 0)
		}
		result[trip.RouteId()] = append(result[trip.RouteId()], trip)
	}
	return result
}

func scheduleForRoute(routeId uuid.UUID, trips []TripScheduleModel) []TripScheduleModel {
	result := schedulesPerRoute(trips)
	if val, ok := result[routeId]; ok {
		return val
	}
	return []TripScheduleModel{}
}

func TestScheduler_ComputeSchedule(t *testing.T) {
	// Setup a fixed reference time for testing
	fixedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Mock time.Now to return our fixed time
	originalTimeNow := timeNow
	timeNow = func() time.Time {
		return fixedTime
	}
	defer func() { timeNow = originalTimeNow }()

	// Create test routes
	routeA := NewBuilder("Route A", 100, 101, 102, 103).
		SetId(uuid.MustParse("11111111-1111-1111-1111-111111111111")).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	routeB := NewBuilder("Route B", 200, 201, 202, 203).
		SetId(uuid.MustParse("22222222-2222-2222-2222-222222222222")).
		SetBoardingWindowDuration(6 * time.Minute).
		SetPreDepartureDuration(3 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(45 * time.Minute).
		Build()

	// Create a shared vessel
	sharedVessel := NewSharedVesselBuilder().
		SetId("shared1").
		SetRouteAID(routeA.Id()).
		SetRouteBID(routeB.Id()).
		SetTurnaroundDelay(5 * time.Minute).
		Build()

	// Test cases
	tests := []struct {
		name                    string
		routes                  []Model
		sharedVessels           []SharedVesselModel
		expectedTripCount       int
		expectedRouteATripCount int
		expectedRouteBTripCount int
	}{
		{
			name:                    "No routes",
			routes:                  []Model{},
			sharedVessels:           []SharedVesselModel{},
			expectedTripCount:       0,
			expectedRouteATripCount: 0,
			expectedRouteBTripCount: 0,
		},
		{
			name:                    "Single route",
			routes:                  []Model{routeA},
			sharedVessels:           []SharedVesselModel{},
			expectedTripCount:       48, // 24 hours / 30 minutes = 48 trips
			expectedRouteATripCount: 48,
			expectedRouteBTripCount: 0,
		},
		{
			name:                    "Multiple routes",
			routes:                  []Model{routeA, routeB},
			sharedVessels:           []SharedVesselModel{},
			expectedTripCount:       80, // 48 for routeA + 32 for routeB (24 hours / 45 minutes = 32 trips)
			expectedRouteATripCount: 48,
			expectedRouteBTripCount: 32,
		},
		{
			name:                    "Shared vessel",
			routes:                  []Model{routeA, routeB},
			sharedVessels:           []SharedVesselModel{sharedVessel},
			expectedTripCount:       136, // 80 from regular routes + 56 from shared vessel
			expectedRouteATripCount: 76,  // 48 from regular route + 28 from shared vessel
			expectedRouteBTripCount: 60,  // 32 from regular route + 28 from shared vessel
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create scheduler with test data
			scheduler := NewScheduler(tt.routes, tt.sharedVessels)

			// Compute schedule
			schedules := scheduler.ComputeSchedule()

			// Assert total trip count
			assert.Equal(t, tt.expectedTripCount, len(schedules), "Total trip count should match expected")

			// Assert route-specific trip counts
			routeASchedules := scheduleForRoute(routeA.Id(), schedules)
			assert.Equal(t, tt.expectedRouteATripCount, len(routeASchedules), "Route A trip count should match expected")

			routeBSchedules := scheduleForRoute(routeB.Id(), schedules)
			assert.Equal(t, tt.expectedRouteBTripCount, len(routeBSchedules), "Route B trip count should match expected")

			// Verify trip times for first route if it exists
			if len(tt.routes) > 0 && len(schedules) > 0 {
				route := tt.routes[0]
				routeSchedules := scheduleForRoute(route.Id(), schedules)

				if len(routeSchedules) > 0 {
					firstTrip := routeSchedules[0]

					// Check that the first trip starts at midnight
					assert.Equal(t, fixedTime, firstTrip.BoardingOpen(), "First trip should start at midnight")

					// Check that boarding closed is after boarding open by the boarding window duration
					expectedBoardingClosed := firstTrip.BoardingOpen().Add(route.BoardingWindowDuration())
					assert.Equal(t, expectedBoardingClosed, firstTrip.BoardingClosed(), "Boarding closed time should be correct")

					// Check that departure is after boarding closed by the pre-departure duration
					expectedDeparture := firstTrip.BoardingClosed().Add(route.PreDepartureDuration())
					assert.Equal(t, expectedDeparture, firstTrip.Departure(), "Departure time should be correct")

					// Check that arrival is after departure by the travel duration
					expectedArrival := firstTrip.Departure().Add(route.TravelDuration())
					assert.Equal(t, expectedArrival, firstTrip.Arrival(), "Arrival time should be correct")

					// If there are at least two trips, check that the second trip starts after the first trip by the cycle interval
					if len(routeSchedules) > 1 {
						secondTrip := routeSchedules[1]
						expectedSecondBoardingOpen := firstTrip.BoardingOpen().Add(route.CycleInterval())
						assert.Equal(t, expectedSecondBoardingOpen, secondTrip.BoardingOpen(), "Second trip should start at the correct time")
					}
				}
			}
		})
	}
}

func TestScheduler_computeRouteSchedule(t *testing.T) {
	// Setup a fixed reference time for testing
	startOfDay := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Create test route
	route := NewBuilder("Test Route", 100, 101, 102, 103).
		SetId(uuid.MustParse("11111111-1111-1111-1111-111111111111")).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	// Create scheduler
	scheduler := NewScheduler([]Model{route}, []SharedVesselModel{})

	// Compute route schedule
	schedules := scheduler.computeRouteSchedule(route, startOfDay, endOfDay)

	// Assert trip count (24 hours / 30 minutes = 48 trips)
	assert.Equal(t, 48, len(schedules), "Should generate 48 trips for a 30-minute cycle over 24 hours")

	// Verify first trip
	if len(schedules) > 0 {
		firstTrip := schedules[0]

		// Check trip ID format
		expectedTripIDPrefix := route.Id().String() + "_"
		assert.True(t, len(firstTrip.TripId()) > len(expectedTripIDPrefix), "Trip ID should have the correct format")
		assert.Equal(t, expectedTripIDPrefix, firstTrip.TripId()[:len(expectedTripIDPrefix)], "Trip ID should start with route ID")

		// Check route ID
		assert.Equal(t, route.Id(), firstTrip.RouteId(), "Trip should be associated with the correct route")

		// Check times
		assert.Equal(t, startOfDay, firstTrip.BoardingOpen(), "First trip should start at midnight")
		assert.Equal(t, startOfDay.Add(route.BoardingWindowDuration()), firstTrip.BoardingClosed(), "Boarding closed time should be correct")
		assert.Equal(t, firstTrip.BoardingClosed().Add(route.PreDepartureDuration()), firstTrip.Departure(), "Departure time should be correct")
		assert.Equal(t, firstTrip.Departure().Add(route.TravelDuration()), firstTrip.Arrival(), "Arrival time should be correct")

		// If there are at least two trips, check the second trip
		if len(schedules) > 1 {
			secondTrip := schedules[1]
			assert.Equal(t, startOfDay.Add(route.CycleInterval()), secondTrip.BoardingOpen(), "Second trip should start at the correct time")
		}
	}
}

func TestScheduler_computeSharedVesselSchedule(t *testing.T) {
	// Setup a fixed reference time for testing
	startOfDay := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Create test routes
	routeA := NewBuilder("Route A", 100, 101, 102, 103).
		SetId(uuid.MustParse("11111111-1111-1111-1111-111111111111")).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute). // This is ignored for shared vessels
		Build()

	routeB := NewBuilder("Route B", 200, 201, 202, 203).
		SetId(uuid.MustParse("22222222-2222-2222-2222-222222222222")).
		SetBoardingWindowDuration(6 * time.Minute).
		SetPreDepartureDuration(3 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(45 * time.Minute). // This is ignored for shared vessels
		Build()

	// Create a shared vessel
	sharedVessel := NewSharedVesselBuilder().
		SetId("shared1").
		SetRouteAID(routeA.Id()).
		SetRouteBID(routeB.Id()).
		SetTurnaroundDelay(5 * time.Minute).
		Build()

	// Create scheduler
	scheduler := NewScheduler([]Model{routeA, routeB}, []SharedVesselModel{sharedVessel})

	// Compute shared vessel schedule
	schedules := scheduler.computeSharedVesselSchedule(sharedVessel, startOfDay, endOfDay)

	// Assert that we have trips
	assert.Greater(t, len(schedules), 0, "Should generate trips for the shared vessel")

	// Verify alternating routes
	if len(schedules) >= 2 {
		firstTrip := schedules[0]
		secondTrip := schedules[1]

		// First trip should be for route A
		assert.Equal(t, routeA.Id(), firstTrip.RouteId(), "First trip should be for route A")

		// Second trip should be for route B
		assert.Equal(t, routeB.Id(), secondTrip.RouteId(), "Second trip should be for route B")

		// Check that the second trip starts after the first trip's arrival plus turnaround delay
		expectedSecondBoardingOpen := firstTrip.Arrival().Add(sharedVessel.TurnaroundDelay())
		assert.Equal(t, expectedSecondBoardingOpen, secondTrip.BoardingOpen(), "Second trip should start after first trip arrival plus turnaround delay")
	}
}
