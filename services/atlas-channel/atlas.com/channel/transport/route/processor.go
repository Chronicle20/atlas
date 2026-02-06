package route

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for route processing
type Processor interface {
	// ByIdProvider retrieves a route by ID
	ByIdProvider(id string) model.Provider[Model]
	// GetById retrieves a route by ID
	GetById(id string) (Model, error)
	// ByStateProvider retrieves a route state by route ID
	ByStateProvider(id string) model.Provider[Model]
	// GetByState retrieves a route state by route ID
	GetByState(id string) (Model, error)
	// ByScheduleProvider retrieves a route schedule by route ID
	ByScheduleProvider(id string) model.Provider[[]TripScheduleModel]
	// GetBySchedule retrieves a route schedule by route ID
	GetBySchedule(id string) ([]TripScheduleModel, error)
	// InTenantProvider retrieves all routes in a tenant
	InTenantProvider() model.Provider[[]Model]
	// GetInTenant retrieves all routes in a tenant
	GetInTenant() ([]Model, error)
	// IsBoatInMap determines if a boat is in the specified map
	IsBoatInMap(mapId _map.Id) (bool, error)
}

// ProcessorImpl is the implementation of Processor
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new route processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// ByIdProvider retrieves a route by ID
func (p *ProcessorImpl) ByIdProvider(id string) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
}

// GetById retrieves a route by ID
func (p *ProcessorImpl) GetById(id string) (Model, error) {
	return p.ByIdProvider(id)()
}

// ByStateProvider retrieves a route state by route ID
func (p *ProcessorImpl) ByStateProvider(id string) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestStateById(id), Extract)
}

// GetByState retrieves a route state by route ID
func (p *ProcessorImpl) GetByState(id string) (Model, error) {
	return p.ByStateProvider(id)()
}

// ByScheduleProvider retrieves a route schedule by route ID
func (p *ProcessorImpl) ByScheduleProvider(id string) model.Provider[[]TripScheduleModel] {
	return requests.SliceProvider[TripScheduleRestModel, TripScheduleModel](p.l, p.ctx)(
		requestScheduleById(id),
		ExtractSchedule,
		model.Filters[TripScheduleModel](),
	)
}

// GetBySchedule retrieves a route schedule by route ID
func (p *ProcessorImpl) GetBySchedule(id string) ([]TripScheduleModel, error) {
	return p.ByScheduleProvider(id)()
}

// InTenantProvider retrieves all routes in a tenant
func (p *ProcessorImpl) InTenantProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInTenant(), Extract, model.Filters[Model]())
}

// GetInTenant retrieves all routes in a tenant
func (p *ProcessorImpl) GetInTenant() ([]Model, error) {
	return p.InTenantProvider()()
}

// IsBoatInMap determines if a boat is in the specified map
// A boat is considered to be in a map if:
// 1. The map ID matches the StartMapId of a route
// 2. The route is in either OpenEntry or LockedEntry state
func (p *ProcessorImpl) IsBoatInMap(mapId _map.Id) (bool, error) {
	p.l.Debugf("Checking if a boat is in map [%d]", mapId)

	// Get all routes for the tenant
	routes, err := p.GetInTenant()
	if err != nil {
		p.l.Errorf("Error retrieving routes for tenant: %v", err)
		return false, err
	}

	// Check if any route matches the criteria
	for _, route := range routes {
		// Check if the map ID matches the StartMapId of the route
		if route.StartMapId() == mapId {
			// Check if the route is in either OpenEntry or LockedEntry state
			state := route.State()
			if state == OpenEntry || state == LockedEntry {
				p.l.Debugf("Found boat in map [%d] for route [%s] in state [%s]",
					mapId, route.Name(), state)
				return true, nil
			}
		}
	}

	p.l.Debugf("No boat found in map [%d]", mapId)
	return false, nil
}
