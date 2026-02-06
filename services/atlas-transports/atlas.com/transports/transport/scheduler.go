package transport

import (
	"time"

	"github.com/google/uuid"
)

var timeNow = time.Now

type Scheduler struct {
	routes        []Model
	sharedVessels []SharedVesselModel
}

func NewScheduler(routes []Model, sharedVessels []SharedVesselModel) *Scheduler {
	return &Scheduler{
		routes:        routes,
		sharedVessels: sharedVessels,
	}
}

func (s *Scheduler) ComputeSchedule() ([]TripScheduleModel, error) {
	now := timeNow().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var schedules []TripScheduleModel

	sharedRouteIds := make(map[uuid.UUID]bool)
	for _, vessel := range s.sharedVessels {
		sharedRouteIds[vessel.RouteAID()] = true
		sharedRouteIds[vessel.RouteBID()] = true
	}

	for _, route := range s.routes {
		if _, isShared := sharedRouteIds[route.Id()]; isShared {
			continue
		}
		routeSchedules, err := s.computeRouteSchedule(route, startOfDay, endOfDay)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, routeSchedules...)
	}

	for _, vessel := range s.sharedVessels {
		vesselSchedules, err := s.computeSharedVesselSchedule(vessel, startOfDay, endOfDay)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, vesselSchedules...)
	}

	return schedules, nil
}

func (s *Scheduler) computeRouteSchedule(route Model, startOfDay, endOfDay time.Time) ([]TripScheduleModel, error) {
	var schedules []TripScheduleModel
	currentTime := startOfDay

	for currentTime.Before(endOfDay) {
		boardingOpen := currentTime
		boardingClosed := boardingOpen.Add(route.BoardingWindowDuration())
		departure := boardingClosed.Add(route.PreDepartureDuration())
		arrival := departure.Add(route.TravelDuration())

		if arrival.Before(endOfDay) {
			schedule, err := NewTripScheduleBuilder().
				SetRouteId(route.Id()).
				SetBoardingOpen(boardingOpen).
				SetBoardingClosed(boardingClosed).
				SetDeparture(departure).
				SetArrival(arrival).
				Build()
			if err != nil {
				return nil, err
			}
			schedules = append(schedules, schedule)
		}
		currentTime = currentTime.Add(route.CycleInterval())
	}

	return schedules, nil
}

func (s *Scheduler) computeSharedVesselSchedule(vessel SharedVesselModel, startOfDay, endOfDay time.Time) ([]TripScheduleModel, error) {
	var schedules []TripScheduleModel

	var routeA, routeB Model
	for _, route := range s.routes {
		if route.Id() == vessel.RouteAID() {
			routeA = route
		} else if route.Id() == vessel.RouteBID() {
			routeB = route
		}
	}

	if routeA.Id() == uuid.Nil || routeB.Id() == uuid.Nil {
		return schedules, nil
	}

	currentTime := startOfDay
	isRouteA := true

	for currentTime.Before(endOfDay) {
		var route Model
		if isRouteA {
			route = routeA
		} else {
			route = routeB
		}

		boardingOpen := currentTime
		boardingClosed := boardingOpen.Add(route.BoardingWindowDuration())
		departure := boardingClosed.Add(route.PreDepartureDuration())
		arrival := departure.Add(route.TravelDuration())

		if arrival.Before(endOfDay) {
			schedule, err := NewTripScheduleBuilder().
				SetRouteId(route.Id()).
				SetBoardingOpen(boardingOpen).
				SetBoardingClosed(boardingClosed).
				SetDeparture(departure).
				SetArrival(arrival).
				Build()
			if err != nil {
				return nil, err
			}
			schedules = append(schedules, schedule)
		}

		currentTime = arrival.Add(vessel.TurnaroundDelay())
		isRouteA = !isRouteA
	}

	return schedules, nil
}
