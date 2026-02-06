package config

import (
	"atlas-transports/instance"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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

func (p *ProcessorImpl) GetInstanceRoutes(tenantId string) ([]instance.RouteModel, error) {
	p.l.Debugf("Fetching instance routes for tenant [%s]", tenantId)
	return requests.SliceProvider[InstanceRouteRestModel, instance.RouteModel](p.l, p.ctx)(requestInstanceRoutes(tenantId), ExtractRoute, model.Filters[instance.RouteModel]())()
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
