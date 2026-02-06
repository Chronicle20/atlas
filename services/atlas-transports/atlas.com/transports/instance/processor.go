package instance

import (
	"atlas-transports/kafka/message"
	it "atlas-transports/kafka/message/instance_transport"
	"atlas-transports/kafka/producer"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	AddTenant(routes []RouteModel)
	ClearTenant() int
	GetRoutes() []RouteModel
	GetRoute(id uuid.UUID) (RouteModel, bool)
	IsTransitMap(mapId _map.Id) bool
	GetRouteByTransitMap(mapId _map.Id) (RouteModel, error)

	StartTransport(mb *message.Buffer) func(characterId uint32, routeId uuid.UUID, f field.Model) error
	StartTransportAndEmit(characterId uint32, routeId uuid.UUID, f field.Model) error

	HandleMapExit(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleMapExitAndEmit(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error

	HandleLogout(mb *message.Buffer) func(characterId uint32, worldId world.Id, channelId channel.Id) error
	HandleLogoutAndEmit(characterId uint32, worldId world.Id, channelId channel.Id) error

	HandleLogin(mb *message.Buffer) func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error
	HandleLoginAndEmit(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error

	TickBoardingExpiration(mb *message.Buffer) error
	TickBoardingExpirationAndEmit() error

	TickArrival(mb *message.Buffer) error
	TickArrivalAndEmit() error

	TickStuckTimeout(mb *message.Buffer) error
	TickStuckTimeoutAndEmit() error

	GracefulShutdown(mb *message.Buffer) error
	GracefulShutdownAndEmit() error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) AddTenant(routes []RouteModel) {
	p.l.Debugf("Adding [%d] instance routes for tenant [%s].", len(routes), p.t.Id())
	getRouteRegistry().AddTenant(p.t, routes)
}

func (p *ProcessorImpl) ClearTenant() int {
	p.l.Debugf("Clearing instance routes for tenant [%s].", p.t.Id())
	return getRouteRegistry().ClearTenant(p.t)
}

func (p *ProcessorImpl) GetRoutes() []RouteModel {
	return getRouteRegistry().GetRoutes(p.t)
}

func (p *ProcessorImpl) GetRoute(id uuid.UUID) (RouteModel, bool) {
	return getRouteRegistry().GetRoute(p.t, id)
}

func (p *ProcessorImpl) IsTransitMap(mapId _map.Id) bool {
	return getRouteRegistry().IsTransitMap(p.t, mapId)
}

func (p *ProcessorImpl) GetRouteByTransitMap(mapId _map.Id) (RouteModel, error) {
	return getRouteRegistry().GetRouteByTransitMap(p.t, mapId)
}

func (p *ProcessorImpl) StartTransport(mb *message.Buffer) func(characterId uint32, routeId uuid.UUID, f field.Model) error {
	return func(characterId uint32, routeId uuid.UUID, f field.Model) error {
		// Double-transport prevention
		cr := getCharacterRegistry()
		if cr.IsInTransport(characterId) {
			p.l.Warnf("Character [%d] is already in an instance transport, rejecting.", characterId)
			return errors.New("character already in transport")
		}

		// Get route
		route, ok := getRouteRegistry().GetRoute(p.t, routeId)
		if !ok {
			return errors.New("instance route not found")
		}

		// Find or create instance
		ir := getInstanceRegistry()
		now := time.Now()
		inst := ir.FindOrCreateInstance(p.t.Id(), route, now)

		// Add character to instance and character registry
		entry := CharacterEntry{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
		}
		ir.AddCharacter(inst.InstanceId(), entry)
		cr.Add(characterId, inst.InstanceId())

		p.l.Infof("Character [%d] boarding instance [%s] for route [%s] (%s). Characters: %d/%d.",
			characterId, inst.InstanceId(), route.Name(), route.Id(), inst.CharacterCount(), route.Capacity())

		// Emit CHANGE_MAP command to transit map with instance
		err := mb.Put(character2EnvCommandTopic, warpToTransitMapProvider(f, characterId, route.TransitMapId(), inst.InstanceId()))
		if err != nil {
			return err
		}

		// Emit STARTED event
		return mb.Put(it.EnvEventTopic, startedEventProvider(f.WorldId(), characterId, route.Id(), inst.InstanceId()))
	}
}

func (p *ProcessorImpl) StartTransportAndEmit(characterId uint32, routeId uuid.UUID, f field.Model) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.StartTransport(mb)(characterId, routeId, f)
	})
}

func (p *ProcessorImpl) HandleMapExit(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, mapId _map.Id, instanceId uuid.UUID, worldId world.Id, channelId channel.Id) error {
		cr := getCharacterRegistry()
		charInstanceId, ok := cr.GetInstanceForCharacter(characterId)
		if !ok {
			return nil // Character not in an instance transport
		}

		// Verify instance matches
		if charInstanceId != instanceId {
			return nil // Not exiting from their transport instance
		}

		ir := getInstanceRegistry()
		inst, ok := ir.GetInstance(charInstanceId)
		if !ok {
			cr.Remove(characterId)
			return nil
		}

		p.l.Infof("Character [%d] exited transit map [%d] instance [%s], cancelling transport.", characterId, mapId, instanceId)

		cr.Remove(characterId)
		empty := ir.RemoveCharacter(charInstanceId, characterId)

		// Emit CANCELLED event
		err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonMapExit))
		if err != nil {
			return err
		}

		if empty {
			p.l.Infof("Instance [%s] is now empty, releasing.", charInstanceId)
			ir.ReleaseInstance(charInstanceId)
		}
		return nil
	}
}

func (p *ProcessorImpl) HandleMapExitAndEmit(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.HandleMapExit(mb)(characterId, mapId, instance, worldId, channelId)
	})
}

func (p *ProcessorImpl) HandleLogout(mb *message.Buffer) func(characterId uint32, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, worldId world.Id, channelId channel.Id) error {
		cr := getCharacterRegistry()
		charInstanceId, ok := cr.GetInstanceForCharacter(characterId)
		if !ok {
			return nil // Character not in an instance transport
		}

		ir := getInstanceRegistry()
		inst, ok := ir.GetInstance(charInstanceId)
		if !ok {
			cr.Remove(characterId)
			return nil
		}

		p.l.Infof("Character [%d] logged out during instance transport [%s], removing from instance.", characterId, charInstanceId)

		cr.Remove(characterId)
		empty := ir.RemoveCharacter(charInstanceId, characterId)

		// Emit CANCELLED event
		err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonLogout))
		if err != nil {
			return err
		}

		if empty {
			p.l.Infof("Instance [%s] is now empty after logout, releasing.", charInstanceId)
			ir.ReleaseInstance(charInstanceId)
		}
		return nil
	}
}

func (p *ProcessorImpl) HandleLogoutAndEmit(characterId uint32, worldId world.Id, channelId channel.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.HandleLogout(mb)(characterId, worldId, channelId)
	})
}

func (p *ProcessorImpl) HandleLogin(mb *message.Buffer) func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
		// Check if the character's login map is a transit map
		route, err := getRouteRegistry().GetRouteByTransitMap(p.t, mapId)
		if err != nil {
			return nil // Not a transit map, nothing to do
		}

		p.l.Infof("Character [%d] logged in at transit map [%d] for route [%s], warping to start map [%d].",
			characterId, mapId, route.Name(), route.StartMapId())

		// Warp character to route start map (uuid.Nil instance)
		return mb.Put(character2EnvCommandTopic, warpToStartMapProvider(worldId, channelId, characterId, route.StartMapId()))
	}
}

func (p *ProcessorImpl) HandleLoginAndEmit(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.HandleLogin(mb)(characterId, mapId, worldId, channelId)
	})
}

func (p *ProcessorImpl) TickBoardingExpiration(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	now := time.Now()

	for _, inst := range ir.GetExpiredBoarding(now) {
		if inst.TenantId() != p.t.Id() {
			continue
		}
		p.l.Infof("Boarding window expired for instance [%s] route [%s], transitioning to InTransit.", inst.InstanceId(), inst.RouteId())
		ir.TransitionToInTransit(inst.InstanceId())
	}
	return nil
}

func (p *ProcessorImpl) TickBoardingExpirationAndEmit() error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.TickBoardingExpiration(mb)
	})
}

func (p *ProcessorImpl) TickArrival(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	cr := getCharacterRegistry()
	now := time.Now()

	for _, inst := range ir.GetExpiredTransit(now) {
		if inst.TenantId() != p.t.Id() {
			continue
		}

		route, ok := getRouteRegistry().GetRoute(p.t, inst.RouteId())
		if !ok {
			p.l.Warnf("Route [%s] not found for arriving instance [%s], releasing.", inst.RouteId(), inst.InstanceId())
			ir.ReleaseInstance(inst.InstanceId())
			continue
		}

		p.l.Infof("Instance [%s] for route [%s] has arrived. Warping %d characters to [%d].",
			inst.InstanceId(), route.Name(), inst.CharacterCount(), route.DestinationMapId())

		characters := inst.Characters()
		for _, entry := range characters {
			err := mb.Put(character2EnvCommandTopic, warpToDestinationProvider(
				entry.WorldId, entry.ChannelId, entry.CharacterId, route.DestinationMapId()))
			if err != nil {
				p.l.WithError(err).Errorf("Error warping character [%d] to destination.", entry.CharacterId)
			}

			// Emit COMPLETED event
			_ = mb.Put(it.EnvEventTopic, completedEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId()))

			cr.Remove(entry.CharacterId)
		}
		ir.ReleaseInstance(inst.InstanceId())
	}
	return nil
}

func (p *ProcessorImpl) TickArrivalAndEmit() error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.TickArrival(mb)
	})
}

func (p *ProcessorImpl) TickStuckTimeout(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	cr := getCharacterRegistry()
	now := time.Now()

	routes := getRouteRegistry().GetRoutes(p.t)
	for _, route := range routes {
		maxLifetime := route.MaxLifetime()
		for _, inst := range ir.GetStuck(now, maxLifetime) {
			if inst.RouteId() != route.Id() || inst.TenantId() != p.t.Id() {
				continue
			}
			p.l.Warnf("Instance [%s] for route [%s] exceeded max lifetime, force-cancelling.", inst.InstanceId(), route.Name())

			characters := inst.Characters()
			for _, entry := range characters {
				_ = mb.Put(character2EnvCommandTopic, warpToStartMapProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, route.StartMapId()))
				_ = mb.Put(it.EnvEventTopic, cancelledEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId(), it.CancelReasonStuck))
				cr.Remove(entry.CharacterId)
			}
			ir.ReleaseInstance(inst.InstanceId())
		}
	}
	return nil
}

func (p *ProcessorImpl) TickStuckTimeoutAndEmit() error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.TickStuckTimeout(mb)
	})
}

func (p *ProcessorImpl) GracefulShutdown(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	cr := getCharacterRegistry()

	for _, inst := range ir.GetAllActive() {
		if inst.TenantId() != p.t.Id() {
			continue
		}

		route, ok := getRouteRegistry().GetRoute(p.t, inst.RouteId())
		if !ok {
			ir.ReleaseInstance(inst.InstanceId())
			continue
		}

		p.l.Infof("Graceful shutdown: warping %d characters from instance [%s] to start map [%d].",
			inst.CharacterCount(), inst.InstanceId(), route.StartMapId())

		characters := inst.Characters()
		for _, entry := range characters {
			_ = mb.Put(character2EnvCommandTopic, warpToStartMapProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, route.StartMapId()))
			cr.Remove(entry.CharacterId)
		}
		ir.ReleaseInstance(inst.InstanceId())
	}
	return nil
}

func (p *ProcessorImpl) GracefulShutdownAndEmit() error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.GracefulShutdown(mb)
	})
}

// character2EnvCommandTopic is the topic environment variable for character commands.
const character2EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"
