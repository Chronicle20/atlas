package instance

import (
	"atlas-transports/kafka/message"
	"atlas-transports/kafka/message/consumable"
	it "atlas-transports/kafka/message/instance_transport"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	AddTenant(routes []RouteModel)
	ClearTenant() int
	GetRoutes() []RouteModel
	GetRoute(id uuid.UUID) (RouteModel, bool)
	IsTransitMap(mapId _map.Id) bool
	GetRouteByTransitMap(mapId _map.Id) (RouteModel, error)
	GetInstancesByRoute(routeId uuid.UUID) []TransportInstance

	StartTransport(mb *message.Buffer) func(characterId uint32, routeId uuid.UUID, f field.Model) error
	StartTransportAndEmit(characterId uint32, routeId uuid.UUID, f field.Model) error

	HandleMapEnter(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error
	HandleMapEnterAndEmit(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error

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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) AddTenant(routes []RouteModel) {
	p.l.Debugf("Adding [%d] instance routes for tenant [%s].", len(routes), p.t.Id())
	getRouteRegistry().AddTenant(p.ctx, routes)
}

func (p *ProcessorImpl) ClearTenant() int {
	p.l.Debugf("Clearing instance routes for tenant [%s].", p.t.Id())
	return getRouteRegistry().ClearTenant(p.ctx)
}

func (p *ProcessorImpl) GetRoutes() []RouteModel {
	return getRouteRegistry().GetRoutes(p.ctx)
}

func (p *ProcessorImpl) GetRoute(id uuid.UUID) (RouteModel, bool) {
	return getRouteRegistry().GetRoute(p.ctx, id)
}

func (p *ProcessorImpl) IsTransitMap(mapId _map.Id) bool {
	return getRouteRegistry().IsTransitMap(p.ctx, mapId)
}

func (p *ProcessorImpl) GetRouteByTransitMap(mapId _map.Id) (RouteModel, error) {
	return getRouteRegistry().GetRouteByTransitMap(p.ctx, mapId)
}

// applyRouteEffects buffers one APPLY_CONSUMABLE_EFFECT per item the route
// declares. It deliberately returns nothing: a missing morph is cosmetic, a
// rejected boarding is not, so a buffer failure is logged and boarding
// continues. A route declaring no effects is a zero-command no-op.
func (p *ProcessorImpl) applyRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32) {
	for _, itemId := range route.EffectItemIds() {
		p.l.Infof("Applying route [%s] effect item [%d] to character [%d].", route.Name(), itemId, characterId)
		if err := mb.Put(consumable.EnvCommandTopic, applyConsumableEffectProvider(worldId, channelId, characterId, itemId)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer apply of effect item [%d] for character [%d] on route [%s].", itemId, characterId, route.Name())
		}
	}
}

// cancelRouteEffects buffers one CANCEL_CONSUMABLE_EFFECT per item the route
// declares, for one character. Like applyRouteEffects it returns nothing: a
// terminal path must always finish releasing its instance even if a command
// cannot be buffered. Leaking a buff is bad; leaking an instance is worse.
//
// A double cancel is harmless — atlas-buffs' Cancel maps a missing buff to
// nil, with no event and no user-visible error — so racing terminal paths
// (portal exit at the same moment the timer fires) need no coordination.
func (p *ProcessorImpl) cancelRouteEffects(mb *message.Buffer, route RouteModel, worldId world.Id, channelId channel.Id, characterId uint32) {
	for _, itemId := range route.EffectItemIds() {
		p.l.Infof("Cancelling route [%s] effect item [%d] for character [%d].", route.Name(), itemId, characterId)
		if err := mb.Put(consumable.EnvCommandTopic, cancelConsumableEffectProvider(worldId, channelId, characterId, itemId)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer cancel of effect item [%d] for character [%d] on route [%s].", itemId, characterId, route.Name())
		}
	}
}

// GetInstancesByRoute returns the live instances for a route. Instances are
// written under the creating tenant's id (StartTransport -> FindOrCreateInstance),
// and the per-route set is tenant-keyed, so the read must use the same
// tenant the request carries - p.t is resolved from the same context
// rest.RegisterHandler installed it into, exactly as every other read in
// this processor relies on.
func (p *ProcessorImpl) GetInstancesByRoute(routeId uuid.UUID) []TransportInstance {
	return getInstanceRegistry().GetInstancesByRoute(p.t.Id(), routeId)
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
		route, ok := getRouteRegistry().GetRoute(p.ctx, routeId)
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
		_, count := ir.AddCharacter(inst.InstanceId(), entry)
		cr.Add(characterId, inst.InstanceId())

		p.l.Infof("Character [%d] boarding instance [%s] for route [%s] (%s). Characters: %d/%d.",
			characterId, inst.InstanceId(), route.Name(), route.Id(), count, route.Capacity())

		// Effect applies are buffered before the CHANGE_MAP command, mirroring
		// the ordering the NPC saga used to guarantee. That ordering is a
		// readability convention, not a guarantee: message.Buffer emits
		// per-topic in Go map iteration order, which is randomised. Correctness
		// does not need one — ApplyConsumableEffect resolves the character's
		// live map at handling time, and an APPLY cannot overtake a later
		// CANCEL because both are keyed by characterId onto a single partition
		// that atlas-consumables consumes serially (maxInFlight defaults to 1).
		p.applyRouteEffects(mb, route, f.WorldId(), f.ChannelId(), characterId)

		// Emit CHANGE_MAP command to transit map with instance
		err := mb.Put(character2EnvCommandTopic, warpToTransitMapProvider(f, characterId, route.TransitMapIds()[0], inst.InstanceId()))
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

func (p *ProcessorImpl) HandleMapEnter(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, mapId _map.Id, instanceId uuid.UUID, worldId world.Id, channelId channel.Id) error {
		isTransit := getRouteRegistry().IsTransitMap(p.ctx, mapId)
		cr := getCharacterRegistry()
		charInstanceId, inTransport := cr.GetInstanceForCharacter(characterId)

		if !isTransit && !inTransport {
			return nil
		}

		if !isTransit && inTransport {
			// Character entered a non-transit map while in transport — cancel
			ir := getInstanceRegistry()
			inst, ok := ir.GetInstance(charInstanceId)
			if !ok {
				cr.Remove(characterId)
				return nil
			}

			p.l.Infof("Character [%d] entered non-transit map [%d] while in transport, cancelling.", characterId, mapId)

			// The route is looked up only for its declared effects; a missing
			// route must not stop the instance from being torn down.
			if route, hasRoute := getRouteRegistry().GetRoute(p.ctx, inst.RouteId()); hasRoute {
				p.cancelRouteEffects(mb, route, worldId, channelId, characterId)
			} else {
				p.l.Warnf("Route [%s] not found while cancelling instance [%s]; character [%d] may retain transit effects.", inst.RouteId(), charInstanceId, characterId)
			}

			cr.Remove(characterId)
			empty := ir.RemoveCharacter(charInstanceId, characterId)

			// A failed event put is logged, not returned: ReleaseInstance below
			// must run regardless (PRD §8 failure isolation).
			if err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonMapExit)); err != nil {
				p.l.WithError(err).Errorf("Unable to buffer CANCELLED event for character [%d]; continuing to instance release.", characterId)
			}

			if empty {
				p.l.Infof("Instance [%s] is now empty, releasing.", charInstanceId)
				ir.ReleaseInstance(charInstanceId)
			}
			return nil
		}

		if isTransit && !inTransport {
			return nil
		}

		// isTransit && inTransport — character moving between transit maps
		// Look up route via character registry, not GetRouteByTransitMap (handles shared transit maps)
		ir := getInstanceRegistry()
		inst, ok := ir.GetInstance(charInstanceId)
		if !ok {
			return nil
		}

		route, ok := getRouteRegistry().GetRoute(p.ctx, inst.RouteId())
		if !ok {
			return nil
		}

		// Verify the entered transit map belongs to this character's route
		if !route.HasTransitMap(mapId) {
			p.l.Warnf("Character [%d] entered transit map [%d] that does not belong to their route [%s].", characterId, mapId, route.Name())
			return nil
		}

		// Emit TRANSIT_ENTERED with remaining time for any transit map entry
		remaining := time.Until(inst.ArrivalAt())
		if remaining < 0 {
			remaining = 0
		}
		remainingSeconds := uint32(remaining.Seconds())
		p.l.Debugf("Character [%d] entered transit map [%d] for route [%s], emitting TRANSIT_ENTERED with [%d]s remaining.", characterId, mapId, route.Name(), remainingSeconds)
		return mb.Put(it.EnvEventTopic, transitEnteredEventProvider(worldId, channelId, characterId, route.Id(), charInstanceId, remainingSeconds, route.TransitMessage()))
	}
}

func (p *ProcessorImpl) HandleMapEnterAndEmit(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.HandleMapEnter(mb)(characterId, mapId, instance, worldId, channelId)
	})
}

func (p *ProcessorImpl) HandleMapExit(mb *message.Buffer) func(characterId uint32, mapId _map.Id, instance uuid.UUID, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, mapId _map.Id, instanceId uuid.UUID, worldId world.Id, channelId channel.Id) error {
		cr := getCharacterRegistry()
		if !cr.IsInTransport(characterId) {
			return nil
		}

		// Cancellation is handled by HandleMapEnter when the character enters a non-transit map.
		// Map exit events don't include the destination, so we can't determine intent here.
		p.l.Debugf("Character [%d] exited map [%d] while in transport, awaiting enter event.", characterId, mapId)
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

		// Best effort (FR-1.6): atlas-buffs does not drop buffs on logout —
		// they carry an expiresAt and are restored with their remaining
		// duration — so without this the player logs back in still morphed.
		// It is not an error if the session is already gone; the command
		// never blocks and never fails the teardown.
		if route, hasRoute := getRouteRegistry().GetRoute(p.ctx, inst.RouteId()); hasRoute {
			p.cancelRouteEffects(mb, route, worldId, channelId, characterId)
		} else {
			p.l.Warnf("Route [%s] not found while cancelling instance [%s] on logout; character [%d] may retain transit effects.", inst.RouteId(), charInstanceId, characterId)
		}

		cr.Remove(characterId)
		empty := ir.RemoveCharacter(charInstanceId, characterId)

		if err := mb.Put(it.EnvEventTopic, cancelledEventProvider(worldId, characterId, inst.RouteId(), charInstanceId, it.CancelReasonLogout)); err != nil {
			p.l.WithError(err).Errorf("Unable to buffer CANCELLED event for character [%d] on logout; continuing to instance release.", characterId)
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
		// Forced-return on disconnect (atlas-maps location.Resolve) ensures the
		// player is never persisted on a transit map. The crash-recovery branch
		// that used to re-warp from a transit map back to route.StartMapId is
		// no longer necessary.
		return nil
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
	now := time.Now()

	for _, inst := range ir.GetExpiredTransit(now) {
		if inst.TenantId() != p.t.Id() {
			continue
		}

		route, ok := getRouteRegistry().GetRoute(p.ctx, inst.RouteId())
		if !ok {
			p.l.Warnf("Route [%s] not found for arriving instance [%s], releasing.", inst.RouteId(), inst.InstanceId())
			ir.ReleaseInstance(inst.InstanceId())
			continue
		}

		p.completeInstance(mb, inst, route)
		ir.ReleaseInstance(inst.InstanceId())
	}
	return nil
}

// completeInstance runs the travel-timer arrival for one instance: cancel each
// character's route effects, warp them out, and emit the terminal event.
//
// A route that declares a forced-return map is one whose transit maps carry a
// client-side timeLimit — running out of flight time is a failure mode there,
// not the delivery mechanism, so the character goes back to the forced-return
// map and the event is CANCELLED/TIMEOUT. Emitting COMPLETED would tell a
// future consumer the character arrived somewhere they never reached. Routes
// without the field (ferries, whose transit maps have no timeLimit at all)
// keep delivering to destinationMapId with COMPLETED, unchanged.
//
// Extracted from TickArrival so the emission is directly testable — the tick's
// clock is time.Now() and cannot be advanced from a test.
func (p *ProcessorImpl) completeInstance(mb *message.Buffer, inst TransportInstance, route RouteModel) {
	cr := getCharacterRegistry()

	forcedReturn := route.ForcedReturnMapId() != 0
	target := route.DestinationMapId()
	if forcedReturn {
		target = route.ForcedReturnMapId()
	}

	p.l.Infof("Instance [%s] for route [%s] has arrived. Warping %d characters to [%d] (forced return: %t).",
		inst.InstanceId(), route.Name(), inst.CharacterCount(), target, forcedReturn)

	for _, entry := range inst.Characters() {
		p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)

		if err := mb.Put(character2EnvCommandTopic, warpToDestinationProvider(
			entry.WorldId, entry.ChannelId, entry.CharacterId, target)); err != nil {
			p.l.WithError(err).Errorf("Error warping character [%d] to [%d].", entry.CharacterId, target)
		}

		if forcedReturn {
			_ = mb.Put(it.EnvEventTopic, cancelledEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId(), it.CancelReasonTimeout))
		} else {
			_ = mb.Put(it.EnvEventTopic, completedEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId()))
		}

		cr.Remove(entry.CharacterId)
	}
}

func (p *ProcessorImpl) TickArrivalAndEmit() error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.TickArrival(mb)
	})
}

func (p *ProcessorImpl) TickStuckTimeout(mb *message.Buffer) error {
	ir := getInstanceRegistry()
	now := time.Now()

	routes := getRouteRegistry().GetRoutes(p.ctx)
	for _, route := range routes {
		maxLifetime := route.MaxLifetime()
		for _, inst := range ir.GetStuck(now, maxLifetime) {
			if inst.RouteId() != route.Id() || inst.TenantId() != p.t.Id() {
				continue
			}
			p.l.Warnf("Instance [%s] for route [%s] exceeded max lifetime, force-cancelling.", inst.InstanceId(), route.Name())
			p.forceCancelInstance(mb, inst, route)
			ir.ReleaseInstance(inst.InstanceId())
		}
	}
	return nil
}

// forceCancelInstance cancels every character's route effects, warps them back
// to the route's start map and emits CANCELLED/STUCK. Extracted from
// TickStuckTimeout so the emission is directly testable — the tick's clock is
// time.Now() and cannot be advanced from a test.
func (p *ProcessorImpl) forceCancelInstance(mb *message.Buffer, inst TransportInstance, route RouteModel) {
	cr := getCharacterRegistry()
	for _, entry := range inst.Characters() {
		p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)
		_ = mb.Put(character2EnvCommandTopic, warpToStartMapProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, route.StartMapId()))
		_ = mb.Put(it.EnvEventTopic, cancelledEventProvider(entry.WorldId, entry.CharacterId, route.Id(), inst.InstanceId(), it.CancelReasonStuck))
		cr.Remove(entry.CharacterId)
	}
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

		route, ok := getRouteRegistry().GetRoute(p.ctx, inst.RouteId())
		if !ok {
			ir.ReleaseInstance(inst.InstanceId())
			continue
		}

		p.l.Infof("Graceful shutdown: warping %d characters from instance [%s] to start map [%d].",
			inst.CharacterCount(), inst.InstanceId(), route.StartMapId())

		characters := inst.Characters()
		for _, entry := range characters {
			p.cancelRouteEffects(mb, route, entry.WorldId, entry.ChannelId, entry.CharacterId)
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
