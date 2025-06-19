package transport

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	AddTenant(routes []Model, sharedVessels []SharedVesselModel) error
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	AllRoutesProvider() model.Provider[[]Model]
	RouteStateByIdProvider(id uuid.UUID) model.Provider[RouteStateModel]
	RouteScheduleByIdProvider(routeId uuid.UUID) model.Provider[[]TripScheduleModel]
	UpdateStates()
}

// ProcessorImpl handles business logic for transport routes
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new processor implementation
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) AddTenant(routes []Model, sharedVessels []SharedVesselModel) error {
	p.l.Debugf("Adding [%d] routes for tenant [%s].", len(routes), p.t.Id())
	getRouteRegistry().AddTenant(p.t, routes)
	getSchedulerRegistry().AddTenant(p.t, NewScheduler(routes, sharedVessels))
	return nil
}

// ByIdProvider returns a provider for a route by id
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return func() (Model, error) {
		for _, route := range getRouteRegistry().GetRoutes(p.t) {
			if route.Id() == id {
				return route, nil
			}
		}
		return Model{}, errors.New("route not found")
	}
}

// RouteStateByIdProvider returns a provider for a route state by route id
func (p *ProcessorImpl) RouteStateByIdProvider(id uuid.UUID) model.Provider[RouteStateModel] {
	return func() (RouteStateModel, error) {
		// Find the route
		_, err := p.ByIdProvider(id)()
		if err != nil {
			return RouteStateModel{}, err
		}

		// Get the state machine for this route
		stateMachine, ok := getRouteRegistry().GetRouteStateMachine(p.t, id)
		if !ok {
			return RouteStateModel{}, errors.New("state machine not found for route")
		}

		// Just return the current state without updating it
		return stateMachine.GetState(), nil
	}
}

// RouteScheduleByIdProvider returns a provider for a route schedule by route id
func (p *ProcessorImpl) RouteScheduleByIdProvider(id uuid.UUID) model.Provider[[]TripScheduleModel] {
	return func() ([]TripScheduleModel, error) {
		// Find the route
		routeProvider := p.ByIdProvider(id)
		_, err := routeProvider()
		if err != nil {
			return nil, err
		}

		// Get the schedule for this route
		routeSchedules := getSchedulerRegistry().Get(p.t).GetScheduleForRoute(id, getSchedulerRegistry().Get(p.t).ComputeSchedule())

		return routeSchedules, nil
	}
}

// AllRoutesProvider returns a provider for all routes
func (p *ProcessorImpl) AllRoutesProvider() model.Provider[[]Model] {
	return func() ([]Model, error) {
		return getRouteRegistry().GetRoutes(p.t), nil
	}
}

// UpdateStates updates the states of all routes
func (p *ProcessorImpl) UpdateStates() {
	now := time.Now()
	for _, stateMachine := range getRouteRegistry().GetStateMachines(p.t) {
		// Call UpdateState but we don't need to use the result here
		_ = stateMachine.UpdateState(now, getSchedulerRegistry().Get(p.t).ComputeSchedule())
	}
}
