package transport

import (
	"atlas-transports/channel"
	"atlas-transports/character"
	"atlas-transports/kafka/message"
	"atlas-transports/kafka/message/transport"
	"atlas-transports/kafka/producer"
	_map "atlas-transports/map"
	"context"
	"errors"
	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
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
	UpdateRoutes() error
	UpdateRouteAndEmit(route Model) error
}

// ProcessorImpl handles business logic for transport routes
type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	t     tenant.Model
	p     producer.Provider
	chanP channel.Processor
	charP character.Processor
	mp    _map.Processor
}

// NewProcessor creates a new processor implementation
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		t:     tenant.MustFromContext(ctx),
		p:     producer.ProviderImpl(l)(ctx),
		chanP: channel.NewProcessor(l, ctx),
		charP: character.NewProcessor(l, ctx),
		mp:    _map.NewProcessor(l, ctx),
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

func (p *ProcessorImpl) UpdateRoutes() error {
	return model.ForEachSlice(p.AllRoutesProvider(), p.UpdateRouteAndEmit, model.ParallelExecute())
}

func (p *ProcessorImpl) UpdateRouteAndEmit(route Model) error {
	return message.Emit(p.p)(model.Flip(p.UpdateRoute)(route))
}

func (p *ProcessorImpl) UpdateRoute(mb *message.Buffer) func(route Model) error {
	return func(route Model) error {
		now := time.Now()
		r, changed := route.UpdateState(now)
		if changed {
			err := getRouteRegistry().UpdateRoute(p.t, r)
			if err != nil {
				p.l.WithError(err).Errorf("Error updating route [%s].", route.Id())
			}
			if r.State() == OpenEntry {
				p.l.Debugf("Transport for route [%s] has arrived.", r.Id())
				err = model.ForEachSlice(model.FixedProvider(p.chanP.GetAll()), func(c channel2.Model) error {
					ff := field.NewBuilder(c.WorldId(), c.Id(), r.EnRouteMapId()).Build()
					tf := field.NewBuilder(c.WorldId(), c.Id(), r.DestinationMapId()).Build()
					p.l.Debugf("Transport for route [%s] is unloading characters in field [%s] to field [%s].", r.Id(), ff.Id(), tf.Id())
					return p.warpTo(mb)(ff, tf)
				}, model.ParallelExecute())
				err = mb.Put(transport.EnvEventTopicStatus, ArrivedStatusEventProvider(r.Id(), r.DestinationMapId()))
				if err != nil {
					p.l.WithError(err).Errorf("Error sending status event for route [%s].", r.Id())
					return err
				}
			} else if r.State() == LockedEntry {
				p.l.Debugf("Transport for route [%s] has locked doors.", r.Id())
			} else if r.State() == InTransit {
				p.l.Debugf("Transport for route [%s] has departed.", r.Id())
				err = model.ForEachSlice(model.FixedProvider(p.chanP.GetAll()), func(c channel2.Model) error {
					ff := field.NewBuilder(c.WorldId(), c.Id(), r.StagingMapId()).Build()
					tf := field.NewBuilder(c.WorldId(), c.Id(), r.EnRouteMapId()).Build()
					p.l.Debugf("Transport for route [%s] is loading characters in field [%s] to field [%s].", r.Id(), ff.Id(), tf.Id())
					return p.warpTo(mb)(ff, tf)
				}, model.ParallelExecute())
				err = mb.Put(transport.EnvEventTopicStatus, DepartedStatusEventProvider(r.Id(), r.StartMapId()))
				if err != nil {
					p.l.WithError(err).Errorf("Error sending status event for route [%s].", r.Id())
					return err
				}
			}
		}
		return nil
	}
}

func (p *ProcessorImpl) warpTo(mb *message.Buffer) func(fromField field.Model, toField field.Model) error {
	return func(ff field.Model, tf field.Model) error {
		var cids []uint32
		cids, err := p.mp.CharacterIdsInMapProvider(ff.WorldId(), ff.ChannelId(), ff.MapId())()
		if err != nil {
			return err
		}
		// TODO perhaps we don't want to warp everyone to a random location
		for _, cid := range cids {
			err = p.charP.WarpRandom(mb)(cid)(tf.Id())
			if err != nil {
				return err
			}
		}
		return nil
	}
}
