package config

import (
	"atlas-transports/instance"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetInstanceRoutes(tenantId string) ([]instance.RouteModel, error)
	LoadConfigurationsForTenant(tenant tenant.Model) ([]instance.RouteModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetInstanceRoutes fetches every instance route configured for a tenant.
// atlas-tenants' GET /tenants/{tenantId}/configurations/instance-routes is
// now paginated (task-117); LoadConfigurationsForTenant (a startup
// per-tenant bootstrap) needs the complete set, so this drains every page
// rather than fetching just the first.
func (p *ProcessorImpl) GetInstanceRoutes(tenantId string) ([]instance.RouteModel, error) {
	p.l.Debugf("Fetching instance routes for tenant [%s]", tenantId)
	return requests.DrainProvider[InstanceRouteRestModel, instance.RouteModel](p.l, p.ctx)(instanceRoutesUrl(tenantId), 250, ExtractRoute, model.Filters[instance.RouteModel]())()
}

func (p *ProcessorImpl) LoadConfigurationsForTenant(tenant tenant.Model) ([]instance.RouteModel, error) {
	tenantId := tenant.Id().String()
	p.l.Infof("Loading instance route configurations for tenant [%s]", tenantId)

	routes, err := p.GetInstanceRoutes(tenantId)
	if err != nil {
		return nil, err
	}

	p.l.Infof("Loaded [%d] instance routes for tenant [%s]", len(routes), tenantId)
	return routes, nil
}
