package transport

import (
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	ByIdProvider(id string) model.Provider[Model]
	RouteStateByIdProvider(id string) model.Provider[RouteStateModel]
	RouteScheduleByIdProvider(routeId string) model.Provider[[]TripScheduleModel]
	UpdateStates()
}

// ProcessorImpl handles business logic for transport routes
type ProcessorImpl struct {
	l             logrus.FieldLogger
	routes        []Model
	sharedVessels []SharedVesselModel
	scheduler     *Scheduler
	stateMachines map[string]*StateMachine
	schedules     []TripScheduleModel
}

// NewProcessor creates a new processor
func NewProcessor(l logrus.FieldLogger, routes []Model, sharedVessels []SharedVesselModel) Processor {
	scheduler := NewScheduler(routes, sharedVessels)
	schedules := scheduler.ComputeSchedule()

	// Create state machines for each route
	stateMachines := make(map[string]*StateMachine)
	for _, route := range routes {
		stateMachines[route.Id()] = NewStateMachine(route)
	}

	return &ProcessorImpl{
		l:             l,
		routes:        routes,
		sharedVessels: sharedVessels,
		scheduler:     scheduler,
		stateMachines: stateMachines,
		schedules:     schedules,
	}
}

// ByIdProvider returns a provider for a route by id
func (p *ProcessorImpl) ByIdProvider(id string) model.Provider[Model] {
	return func() (Model, error) {
		for _, route := range p.routes {
			if route.Id() == id {
				return route, nil
			}
		}
		return Model{}, errors.New("route not found")
	}
}

// RouteStateByIdProvider returns a provider for a route state by route id
func (p *ProcessorImpl) RouteStateByIdProvider(id string) model.Provider[RouteStateModel] {
	return func() (RouteStateModel, error) {
		// Find the route
		routeProvider := p.ByIdProvider(id)
		_, err := routeProvider()
		if err != nil {
			return RouteStateModel{}, err
		}

		// Get the state machine for this route
		stateMachine, ok := p.stateMachines[id]
		if !ok {
			return RouteStateModel{}, errors.New("state machine not found for route")
		}

		// Update the state based on the current time
		now := time.Now()
		state := stateMachine.UpdateState(now, p.schedules)

		return state, nil
	}
}

// RouteScheduleByIdProvider returns a provider for a route schedule by route id
func (p *ProcessorImpl) RouteScheduleByIdProvider(routeId string) model.Provider[[]TripScheduleModel] {
	return func() ([]TripScheduleModel, error) {
		// Find the route
		routeProvider := p.ByIdProvider(routeId)
		_, err := routeProvider()
		if err != nil {
			return nil, err
		}

		// Get the schedule for this route
		routeSchedules := p.scheduler.GetScheduleForRoute(routeId, p.schedules)

		return routeSchedules, nil
	}
}

// UpdateStates updates the states of all routes
func (p *ProcessorImpl) UpdateStates() {
	now := time.Now()
	for _, stateMachine := range p.stateMachines {
		stateMachine.UpdateState(now, p.schedules)
	}
}
