package config

import (
	"atlas-transports/transport"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for configuration operations
type Processor interface {
	// GetRoutes returns all routes for a tenant
	GetRoutes(tenantId string) ([]transport.Model, error)

	// GetVessels returns all vessels for a tenant
	GetVessels(tenantId string) ([]transport.SharedVesselModel, error)

	// LoadConfigurationsForTenant loads all configurations for a tenant and returns routes and vessels
	LoadConfigurationsForTenant(tenant tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new processor implementation
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetRoutes returns all routes for a tenant
func (p *ProcessorImpl) GetRoutes(tenantId string) ([]transport.Model, error) {
	p.l.Debugf("Fetching routes for tenant [%s]", tenantId)
	return requests.SliceProvider[RouteRestModel, transport.Model](p.l, p.ctx)(requestRoutes(tenantId), ExtractRoute, model.Filters[transport.Model]())()
}

// GetVessels returns all vessels for a tenant
func (p *ProcessorImpl) GetVessels(tenantId string) ([]transport.SharedVesselModel, error) {
	p.l.Debugf("Fetching vessels for tenant [%s]", tenantId)
	return requests.SliceProvider[VesselRestModel, transport.SharedVesselModel](p.l, p.ctx)(requestVessels(tenantId), ExtractVessel, model.Filters[transport.SharedVesselModel]())()
}

// LoadConfigurationsForTenant loads all configurations for a tenant and returns routes and vessels
func (p *ProcessorImpl) LoadConfigurationsForTenant(tenant tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error) {
	tenantId := tenant.Id().String()
	p.l.Infof("Loading configurations for tenant [%s]", tenantId)

	routes, err := p.GetRoutes(tenantId)
	if err != nil {
		return nil, nil, err
	}

	vessels, err := p.GetVessels(tenantId)
	if err != nil {
		return nil, nil, err
	}

	p.l.Infof("Loaded [%d] routes and [%d] vessels for tenant [%s]", len(routes), len(vessels), tenantId)
	return routes, vessels, nil
}
