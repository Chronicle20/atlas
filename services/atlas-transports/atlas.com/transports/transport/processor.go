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
	map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	AddTenant(routes []Model, sharedVessels []SharedVesselModel) error
	ClearTenant() int
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	ByStartMapProvider(mapId map2.Id) model.Provider[Model]
	AllRoutesProvider() model.Provider[[]Model]
	UpdateRoutes() error
	UpdateRouteAndEmit(route Model) error
	WarpToRouteStartMapOnLogout(mb *message.Buffer) func(characterId uint32, f field.Model) error
	WarpToRouteStartMapOnLogoutAndEmit(characterId uint32, f field.Model) error
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

// ClearTenant removes all routes for the current tenant and returns the count of deleted routes
func (p *ProcessorImpl) ClearTenant() int {
	p.l.Debugf("Clearing routes for tenant [%s].", p.t.Id())
	return getRouteRegistry().ClearTenant(p.t)
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

// ByStartMapProvider returns a provider for a route by start map id
func (p *ProcessorImpl) ByStartMapProvider(mapId map2.Id) model.Provider[Model] {
	return func() (Model, error) {
		return getRouteRegistry().GetRouteByStartMap(p.t, mapId)
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
			if r.State() == AwaitingReturn {
				p.l.Infof("Transport for route [%s] has arrived at [%d].", r.Id(), r.DestinationMapId())
				for _, enRouteMapId := range r.EnRouteMapIds() {
					err = model.ForEachSlice(model.FixedProvider(p.chanP.GetAll()), func(c channel2.Model) error {
						ff := field.NewBuilder(c.WorldId(), c.Id(), enRouteMapId).Build()
						tf := field.NewBuilder(c.WorldId(), c.Id(), r.DestinationMapId()).Build()
						return p.warpTo(mb)(ff, tf)
					}, model.ParallelExecute())
					if err != nil {
						p.l.WithError(err).Errorf("Error warping characters from enroute map [%d] to destination map [%d].", enRouteMapId, r.DestinationMapId())
						return err
					}
				}
			}
			if r.State() == OpenEntry {
				err = mb.Put(transport.EnvEventTopicStatus, ArrivedStatusEventProvider(r.Id(), r.ObservationMapId()))
				if err != nil {
					p.l.WithError(err).Errorf("Error sending status event for route [%s].", r.Id())
					return err
				}
			} else if r.State() == LockedEntry {
				p.l.Infof("Transport for route [%s] has locked doors at [%d].", r.Id(), r.StagingMapId())
			} else if r.State() == InTransit {
				p.l.Infof("Transport for route [%s] has departed [%d].", r.Id(), r.StagingMapId())
				err = model.ForEachSlice(model.FixedProvider(p.chanP.GetAll()), func(c channel2.Model) error {
					ff := field.NewBuilder(c.WorldId(), c.Id(), r.StagingMapId()).Build()
					tf := field.NewBuilder(c.WorldId(), c.Id(), r.EnRouteMapIds()[0]).Build()
					return p.warpTo(mb)(ff, tf)
				}, model.ParallelExecute())
				if err != nil {
					p.l.WithError(err).Errorf("Error warping characters from staging map [%d] to enroute map.", r.StagingMapId())
					return err
				}
				err = mb.Put(transport.EnvEventTopicStatus, DepartedStatusEventProvider(r.Id(), r.ObservationMapId()))
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
		cp := p.mp.CharacterIdsInMapProvider(ff.WorldId(), ff.ChannelId(), ff.MapId())
		return model.ForEachSlice(cp, func(characterId uint32) error {
			p.l.Infof("Warping character [%d] from map [%d] to map [%d].", characterId, ff.MapId(), tf.MapId())
			return p.charP.WarpRandom(mb)(characterId)(tf.Id())
		})
	}
}

func (p *ProcessorImpl) WarpToRouteStartMapOnLogout(mb *message.Buffer) func(characterId uint32, f field.Model) error {
	return func(characterId uint32, f field.Model) error {
		// Get all routes for the tenant
		routes, err := p.AllRoutesProvider()()
		if err != nil {
			p.l.WithError(err).Error("Failed to get routes for tenant")
			return err
		}

		for _, route := range routes {
			var mapIds []map2.Id
			mapIds = append(mapIds, route.StagingMapId())
			mapIds = append(mapIds, route.EnRouteMapIds()...)

			for _, routeMapId := range mapIds {
				if routeMapId != f.MapId() {
					continue
				}
				p.l.Debugf("Character [%d] logged out in map [%d] for route [%s], warping to start map [%d]", characterId, f.MapId(), route.Id(), route.StartMapId())
				tf := field.NewBuilder(f.WorldId(), f.ChannelId(), route.StartMapId()).Build()
				return p.charP.WarpRandom(mb)(characterId)(tf.Id())
			}
		}
		return nil
	}
}

func (p *ProcessorImpl) WarpToRouteStartMapOnLogoutAndEmit(characterId uint32, f field.Model) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.WarpToRouteStartMapOnLogout(mb)(characterId, f)
	})
}
