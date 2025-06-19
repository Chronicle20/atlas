package transport

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

// Variable to allow mocking time.Now for testing
var timeNow = time.Now

// Scheduler computes trip schedules for transport routes
type Scheduler struct {
	routes        []Model
	sharedVessels []SharedVesselModel
}

// NewScheduler creates a new scheduler
func NewScheduler(routes []Model, sharedVessels []SharedVesselModel) *Scheduler {
	return &Scheduler{
		routes:        routes,
		sharedVessels: sharedVessels,
	}
}

// ComputeSchedule computes the trip schedule for all routes for the current UTC day
func (s *Scheduler) ComputeSchedule() []TripScheduleModel {
	now := timeNow().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var schedules []TripScheduleModel

	// Process regular routes
	for _, route := range s.routes {
		routeSchedules := s.computeRouteSchedule(route, startOfDay, endOfDay)
		schedules = append(schedules, routeSchedules...)
	}

	// Process shared vessels
	for _, vessel := range s.sharedVessels {
		vesselSchedules := s.computeSharedVesselSchedule(vessel, startOfDay, endOfDay)
		schedules = append(schedules, vesselSchedules...)
	}

	return schedules
}

// computeRouteSchedule computes the trip schedule for a single route
func (s *Scheduler) computeRouteSchedule(route Model, startOfDay, endOfDay time.Time) []TripScheduleModel {
	var schedules []TripScheduleModel

	// Start at midnight and compute trips until the end of the day
	currentTime := startOfDay
	for currentTime.Before(endOfDay) {
		// Calculate trip times
		boardingOpen := currentTime
		boardingClosed := boardingOpen.Add(route.BoardingWindowDuration())
		departure := boardingClosed.Add(route.PreDepartureDuration())
		arrival := departure.Add(route.TravelDuration())

		// Only include trips that are fully contained within the day
		if arrival.Before(endOfDay) {
			// Create trip ID in the format: {routeID}_{departureTimestamp}
			tripID := fmt.Sprintf("%s_%s", route.Id(), departure.Format("20060102T150405"))

			// Create trip schedule
			schedule := NewTripScheduleBuilder().
				SetTripID(tripID).
				SetRouteID(route.Id()).
				SetBoardingOpen(boardingOpen).
				SetBoardingClosed(boardingClosed).
				SetDeparture(departure).
				SetArrival(arrival).
				Build()

			schedules = append(schedules, schedule)
		}

		// Move to the next cycle
		currentTime = currentTime.Add(route.CycleInterval())
	}

	return schedules
}

// computeSharedVesselSchedule computes the trip schedule for a shared vessel
func (s *Scheduler) computeSharedVesselSchedule(vessel SharedVesselModel, startOfDay, endOfDay time.Time) []TripScheduleModel {
	var schedules []TripScheduleModel

	// Find the routes for this shared vessel
	var routeA, routeB Model
	for _, route := range s.routes {
		if route.Id() == vessel.RouteAID() {
			routeA = route
		} else if route.Id() == vessel.RouteBID() {
			routeB = route
		}
	}

	// If either route is not found, return empty schedule
	if routeA.Id() == uuid.Nil || routeB.Id() == uuid.Nil {
		return schedules
	}

	// Start at midnight and compute trips until the end of the day
	currentTime := startOfDay
	isRouteA := true // Start with route A

	for currentTime.Before(endOfDay) {
		var route Model
		if isRouteA {
			route = routeA
		} else {
			route = routeB
		}

		// Calculate trip times
		boardingOpen := currentTime
		boardingClosed := boardingOpen.Add(route.BoardingWindowDuration())
		departure := boardingClosed.Add(route.PreDepartureDuration())
		arrival := departure.Add(route.TravelDuration())

		// Only include trips that are fully contained within the day
		if arrival.Before(endOfDay) {
			// Create trip ID in the format: {routeID}_{departureTimestamp}
			tripID := fmt.Sprintf("%s_%s", route.Id(), departure.Format("20060102T150405"))

			// Create trip schedule
			schedule := NewTripScheduleBuilder().
				SetTripID(tripID).
				SetRouteID(route.Id()).
				SetBoardingOpen(boardingOpen).
				SetBoardingClosed(boardingClosed).
				SetDeparture(departure).
				SetArrival(arrival).
				Build()

			schedules = append(schedules, schedule)
		}

		// Move to the next cycle, alternating between routes
		currentTime = arrival.Add(vessel.TurnaroundDelay())
		isRouteA = !isRouteA
	}

	return schedules
}

// GetScheduleForRoute returns the trip schedule for a specific route
func (s *Scheduler) GetScheduleForRoute(routeID uuid.UUID, schedules []TripScheduleModel) []TripScheduleModel {
	var routeSchedules []TripScheduleModel
	for _, schedule := range schedules {
		if schedule.RouteID() == routeID {
			routeSchedules = append(routeSchedules, schedule)
		}
	}
	return routeSchedules
}
