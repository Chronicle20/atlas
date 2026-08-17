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
	GetInstanceRoutes(t tenant.Model) ([]instance.RouteModel, error)
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
func (p *ProcessorImpl) GetInstanceRoutes(t tenant.Model) ([]instance.RouteModel, error) {
	p.l.Debugf("Fetching instance routes for tenant [%s]", t.Id())
	url, err := instanceRoutesUrl(p.ctx, t.Id().String())
	if err != nil {
		return nil, err
	}
	return requests.DrainProvider[InstanceRouteRestModel, instance.RouteModel](p.l, p.ctx)(url, 250, ExtractRouteFor(p.l, t), model.Filters[instance.RouteModel]())()
}

func (p *ProcessorImpl) LoadConfigurationsForTenant(t tenant.Model) ([]instance.RouteModel, error) {
	p.l.Infof("Loading instance route configurations for tenant [%s]", t.Id())

	routes, err := p.GetInstanceRoutes(t)
	if err != nil {
		return nil, err
	}

	p.l.Infof("Loaded [%d] instance routes for tenant [%s]", len(routes), t.Id())
	return routes, nil
}
