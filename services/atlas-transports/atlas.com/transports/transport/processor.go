package transport

import (
	"atlas-transports/channel"
	"atlas-transports/kafka/message/transport"
	"atlas-transports/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	AddTenant(routes []Model, sharedVessels []SharedVesselModel) error
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	AllRoutesProvider() model.Provider[[]Model]
	UpdateStates() error
}

// ProcessorImpl handles business logic for transport routes
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	cp  channel.Processor
}

// NewProcessor creates a new processor implementation
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		cp:  channel.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) AddTenant(distinctRoutes []Model, sharedVessels []SharedVesselModel) error {
	p.l.Debugf("Adding [%d] routes for tenant [%s].", len(distinctRoutes), p.t.Id())
	routeMap := make(map[uuid.UUID]Model)
	for _, route := range distinctRoutes {
		routeMap[route.Id()] = route
	}
	schedules := NewScheduler(distinctRoutes, sharedVessels).ComputeSchedule()
	for _, schedule := range schedules {
		if route, ok := routeMap[schedule.RouteId()]; ok {
			routeMap[route.Id()] = route.Builder().AddToSchedule(schedule).Build()
		}
	}
	scheduledRoutes := make([]Model, 0)
	for _, route := range routeMap {
		scheduledRoutes = append(scheduledRoutes, route)
	}

	getRouteRegistry().AddTenant(p.t, scheduledRoutes)
	return nil
}

// ByIdProvider returns a provider for a route by id
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return func() (Model, error) {
		m, ok := getRouteRegistry().GetRoute(p.t, id)
		if !ok {
			return Model{}, errors.New("route not found")
		}
		return m, nil
	}
}

// AllRoutesProvider returns a provider for all routes
func (p *ProcessorImpl) AllRoutesProvider() model.Provider[[]Model] {
	return func() ([]Model, error) {
		return getRouteRegistry().GetRoutes(p.t)
	}
}

// UpdateStates updates the states of all routes
func (p *ProcessorImpl) UpdateStates() error {
	now := time.Now()

	routes, err := getRouteRegistry().GetRoutes(p.t)
	if err != nil {
		return err
	}
	for _, route := range routes {
		r, changed := route.UpdateState(now)
		if changed {
			err = getRouteRegistry().UpdateRoute(p.t, r)
			if err != nil {
				p.l.WithError(err).Errorf("Error updating route [%s].", route.Id())
			}
			var messageProvider model.Provider[[]kafka.Message]
			if r.State() == OpenEntry {
				p.l.Debugf("Transport for route [%s] has arrived.", r.Id())
				for _, c := range p.cp.GetAll() {
					ff := field.NewBuilder(c.WorldId(), c.Id(), r.EnRouteMapId()).Build()
					tf := field.NewBuilder(c.WorldId(), c.Id(), r.DestinationMapId()).Build()
					p.l.Debugf("Transport for route [%s] is unloading characters in field [%s] to field [%s].", r.Id(), ff.Id(), tf.Id())
				}
				messageProvider = ArrivedStatusEventProvider(r.Id(), r.DestinationMapId())
			} else if r.State() == LockedEntry {
				p.l.Debugf("Transport for route [%s] has locked doors.", r.Id())
			} else if r.State() == InTransit {
				p.l.Debugf("Transport for route [%s] has departed.", r.Id())
				for _, c := range p.cp.GetAll() {
					ff := field.NewBuilder(c.WorldId(), c.Id(), r.StagingMapId()).Build()
					tf := field.NewBuilder(c.WorldId(), c.Id(), r.EnRouteMapId()).Build()
					p.l.Debugf("Transport for route [%s] is loading characters in field [%s] to field [%s].", r.Id(), ff.Id(), tf.Id())
				}
				messageProvider = DepartedStatusEventProvider(r.Id(), r.StartMapId())
			}
			if messageProvider != nil {
				_ = producer.ProviderImpl(p.l)(p.ctx)(transport.EnvEventTopicStatus)(messageProvider)
			}
		}
	}
	return nil
}
