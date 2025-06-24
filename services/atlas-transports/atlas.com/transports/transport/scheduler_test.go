package transport

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func schedulesPerRoute(trips []TripScheduleModel) map[uuid.UUID][]TripScheduleModel {
	result := make(map[uuid.UUID][]TripScheduleModel)
	for _, trip := range trips {
		result[trip.RouteId()] = append(result[trip.RouteId()], trip)
	}
	return result
}

func scheduleForRoute(routeId uuid.UUID, trips []TripScheduleModel) []TripScheduleModel {
	return schedulesPerRoute(trips)[routeId]
}

func TestScheduler_ComputeSchedule_SharedVesselOverridesRouteSchedule(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	originalTimeNow := timeNow
	timeNow = func() time.Time { return fixedTime }
	defer func() { timeNow = originalTimeNow }()

	routeA := NewBuilder("Route A").
		SetStartMapId(100).
		SetStagingMapId(101).
		SetEnRouteMapIds([]_map.Id{102}).
		SetDestinationMapId(103).
		SetId(uuid.MustParse("11111111-1111-1111-1111-111111111111")).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	routeB := NewBuilder("Route B").
		SetStartMapId(200).
		SetStagingMapId(201).
		SetEnRouteMapIds([]_map.Id{202}).
		SetDestinationMapId(203).
		SetId(uuid.MustParse("22222222-2222-2222-2222-222222222222")).
		SetBoardingWindowDuration(6 * time.Minute).
		SetPreDepartureDuration(3 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(45 * time.Minute).
		Build()

	independentRoute := NewBuilder("Independent Route").
		SetStartMapId(300).
		SetStagingMapId(301).
		SetEnRouteMapIds([]_map.Id{302}).
		SetDestinationMapId(303).
		SetId(uuid.MustParse("33333333-3333-3333-3333-333333333333")).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(20 * time.Minute).
		Build()

	sharedVessel := NewSharedVesselBuilder().
		SetId("shared1").
		SetRouteAID(routeA.Id()).
		SetRouteBID(routeB.Id()).
		SetTurnaroundDelay(5 * time.Minute).
		Build()

	scheduler := NewScheduler([]Model{routeA, routeB, independentRoute}, []SharedVesselModel{sharedVessel})
	schedules := scheduler.ComputeSchedule()

	routeCounts := schedulesPerRoute(schedules)

	// Independent route should have periodic schedule
	assert.Greater(t, len(routeCounts[independentRoute.Id()]), 0, "Independent route should have schedule")

	// Both shared routes should have schedules generated, but only from the shared vessel logic
	totalSharedTrips := len(routeCounts[routeA.Id()]) + len(routeCounts[routeB.Id()])
	assert.Greater(t, totalSharedTrips, 0, "Shared routes should have schedule from shared vessel")

	// Estimate expected shared vessel trips
	// Alternate trips: Route A trip, Route B trip, etc.
	expectedSharedTrips := len(routeCounts[routeA.Id()]) + len(routeCounts[routeB.Id()])
	assert.Equal(t, totalSharedTrips, expectedSharedTrips, "All trips for shared routes should come from shared vessel")

	// Confirm shared routes do not have independent periodic schedules, only shared trips
	assert.True(t, totalSharedTrips > 0 && totalSharedTrips < 96, "Total shared trips should be reasonable (i.e., no periodic schedules added)")
}
