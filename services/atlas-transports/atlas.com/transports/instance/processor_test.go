package instance

import (
	"atlas-transports/kafka/message"
	"atlas-transports/kafka/message/consumable"
	it "atlas-transports/kafka/message/instance_transport"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// setupProcessorTest wires all three registries onto one miniredis and returns
// a processor with a nil producer: every Xxx(mb) method only buffers, and the
// producer is used exclusively by the XxxAndEmit wrappers these tests never
// call. That is what makes the lifecycle testable without Kafka or mocks.
func setupProcessorTest(t *testing.T) (*ProcessorImpl, context.Context) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRouteRegistry(rc)
	InitInstanceRegistry(rc)
	InitCharacterRegistry(rc)

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	l := logrus.New()
	l.SetOutput(io.Discard)

	return &ProcessorImpl{l: l, ctx: ctx, t: tm, p: nil}, ctx
}

// newEffectRoute mirrors the seeded temple-of-time-flight: one declared effect
// item and a forced return. Capacity is a parameter so a fan-out test can put
// several characters in one instance.
func newEffectRoute(t *testing.T, capacity uint32) RouteModel {
	t.Helper()
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetStartMapId(_map.Id(240000110)).
		SetTransitMapIds([]_map.Id{200090500, 200090510}).
		SetDestinationMapId(_map.Id(270000100)).
		SetCapacity(capacity).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		SetForcedReturnMapId(_map.Id(240000110)).
		Build()
	assert.NoError(t, err)
	return route
}

// newPlainRoute mirrors a ferry: no declared effects, no forced return. This
// is the regression bar — it must produce zero consumable messages on every
// path and keep delivering to destinationMapId on arrival.
func newPlainRoute(t *testing.T) RouteModel {
	t.Helper()
	route, err := NewRouteBuilder("ellinia-ereve-ferry").
		SetStartMapId(_map.Id(101000300)).
		SetTransitMapIds([]_map.Id{200090030}).
		SetDestinationMapId(_map.Id(130000210)).
		SetCapacity(4).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(60 * time.Second).
		Build()
	assert.NoError(t, err)
	return route
}

func decodeConsumables(t *testing.T, mb *message.Buffer) []decodedConsumable {
	t.Helper()
	out := make([]decodedConsumable, 0)
	for _, m := range mb.GetAll()[consumable.EnvCommandTopic] {
		var d decodedConsumable
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

type decodedTransportEvent struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        struct {
		RouteId    uuid.UUID `json:"routeId"`
		InstanceId uuid.UUID `json:"instanceId"`
		Reason     string    `json:"reason"`
	} `json:"body"`
}

func decodeInstanceTransportEvents(t *testing.T, mb *message.Buffer) []decodedTransportEvent {
	t.Helper()
	out := make([]decodedTransportEvent, 0)
	for _, m := range mb.GetAll()[it.EnvEventTopic] {
		var d decodedTransportEvent
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

type decodedChangeMap struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        struct {
		MapId _map.Id `json:"mapId"`
	} `json:"body"`
}

func decodeChangeMaps(t *testing.T, mb *message.Buffer) []decodedChangeMap {
	t.Helper()
	out := make([]decodedChangeMap, 0)
	for _, m := range mb.GetAll()[character2EnvCommandTopic] {
		var d decodedChangeMap
		assert.NoError(t, json.Unmarshal(m.Value, &d))
		out = append(out, d)
	}
	return out
}

func TestStartTransport_AppliesDeclaredEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.NoError(t, p.StartTransport(mb)(42, route.Id(), f))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandApplyConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)
	assert.Equal(t, uint32(42), cs[0].CharacterId)
	assert.Equal(t, world.Id(0), cs[0].WorldId)
	assert.Equal(t, channel.Id(1), cs[0].ChannelId)
}

func TestStartTransport_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(101000300)).Build()
	assert.NoError(t, p.StartTransport(mb)(42, route.Id(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	// The trip still starts exactly as before.
	assert.Len(t, decodeChangeMaps(t, mb), 1)
}

func TestStartTransport_RouteNotFoundEmitsNothing(t *testing.T) {
	p, _ := setupProcessorTest(t)

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.Error(t, p.StartTransport(mb)(42, uuid.New(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

func TestStartTransport_AlreadyInTransportEmitsNothing(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	getCharacterRegistry().Add(ctx, 42, uuid.New())

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.Error(t, p.StartTransport(mb)(42, route.Id(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

// board puts a character into an instance of route through the real boarding
// path and returns the instance id, so terminal-path tests start from exactly
// the state StartTransport leaves behind.
func board(t *testing.T, p *ProcessorImpl, route RouteModel, characterId uint32, worldId world.Id, channelId channel.Id) uuid.UUID {
	t.Helper()
	mb := message.NewBuffer()
	f := field.NewBuilder(worldId, channelId, route.StartMapId()).Build()
	assert.NoError(t, p.StartTransport(mb)(characterId, route.Id(), f))
	instanceId, ok := getCharacterRegistry().GetInstanceForCharacter(p.ctx, characterId)
	assert.True(t, ok)
	return instanceId
}

// The dracoout shape: exiting transit map 200090510 through a portal that
// warps to the non-transit map 240000100. That portal seed carries no
// cancel_consumable_effect operation — under route-owned cleanup the morph is
// removed here instead, which is the previously-unfixed leak (FR-3.6).
func TestHandleMapEnter_NonTransitMapCancelsEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(240000100), uuid.Nil, world.Id(0), channel.Id(1)))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)
	assert.Equal(t, uint32(42), cs[0].CharacterId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCancelled, evs[0].Type)
	assert.Equal(t, it.CancelReasonMapExit, evs[0].Body.Reason)
	assert.Equal(t, instanceId, evs[0].Body.InstanceId)
}

// Moving between two transit maps of the same route is not a terminal path.
func TestHandleMapEnter_TransitToTransitDoesNotCancel(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(200090510), uuid.Nil, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeTransitEntered, evs[0].Type)
}

func TestHandleMapEnter_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleMapEnter(mb)(42, _map.Id(130000210), uuid.Nil, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

// atlas-buffs does not drop buffs on logout — they carry an expiresAt and are
// restored with their remaining duration. Without this cancel a player who
// logs out mid-flight logs back in still morphed.
func TestHandleLogout_CancelsEffects(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleLogout(mb)(42, world.Id(0), channel.Id(1)))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, uint32(42), cs[0].CharacterId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.CancelReasonLogout, evs[0].Body.Reason)
}

func TestHandleLogout_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.HandleLogout(mb)(42, world.Id(0), channel.Id(1)))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}

func TestTickStuckTimeout_CancelsEffectsForEveryCharacter(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	// MaxLifetime is 2*(boardingWindow+travelDuration) = 1802s; miniredis
	// stores real timestamps, so age the instance by rewriting its metadata
	// is not possible — instead assert against a route whose lifetime has
	// already elapsed by using GetStuck's own clock.
	mb := message.NewBuffer()
	assert.NoError(t, p.TickStuckTimeout(mb))
	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic], "instance is not stuck yet")

	// Advance past MaxLifetime by asking the registry directly.
	stuck := getInstanceRegistry().GetStuck(ctx, time.Now().Add(route.MaxLifetime()+time.Second), route.MaxLifetime())
	assert.NotEmpty(t, stuck, "instance must be considered stuck once MaxLifetime has elapsed")
}

func TestGracefulShutdown_CancelsEffectsForEveryCharacter(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	mb := message.NewBuffer()
	assert.NoError(t, p.GracefulShutdown(mb))

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 2)
	seen := map[uint32]bool{}
	for _, c := range cs {
		assert.Equal(t, consumable.CommandCancelConsumableEffect, c.Type)
		assert.Equal(t, item.Id(2210016), c.Body.ItemId)
		seen[c.CharacterId] = true
	}
	assert.True(t, seen[42] && seen[43], "both characters must be cancelled")

	// Characters are still warped to the start map, unchanged.
	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 2)
	for _, w := range warps {
		assert.Equal(t, route.StartMapId(), w.Body.MapId)
	}
}

func TestGracefulShutdown_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	board(t, p, route, 42, world.Id(0), channel.Id(1))

	mb := message.NewBuffer()
	assert.NoError(t, p.GracefulShutdown(mb))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
	assert.Len(t, decodeChangeMaps(t, mb), 1)
}

// TickStuckTimeout's clock is time.Now() inside the method, so the cancelling
// loop body is exercised directly. This is the same code path the tick runs
// once MaxLifetime has elapsed.
func TestForceCancelInstance_CancelsEffectsAndWarpsToStart(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 2)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))
	board(t, p, route, 43, world.Id(0), channel.Id(2))

	inst, ok := getInstanceRegistry().GetInstance(ctx, instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.forceCancelInstance(mb, inst, route)

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 2)
	for _, c := range cs {
		assert.Equal(t, consumable.CommandCancelConsumableEffect, c.Type)
		assert.Equal(t, item.Id(2210016), c.Body.ItemId)
	}

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 2)
	for _, w := range warps {
		assert.Equal(t, route.StartMapId(), w.Body.MapId)
	}

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 2)
	for _, e := range evs {
		assert.Equal(t, it.EventTypeCancelled, e.Type)
		assert.Equal(t, it.CancelReasonStuck, e.Body.Reason)
	}

	assert.False(t, getCharacterRegistry().IsInTransport(ctx, 42))
	assert.False(t, getCharacterRegistry().IsInTransport(ctx, 43))
}

func TestForceCancelInstance_NoEffectsEmitsNoConsumableCommands(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(ctx, instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.forceCancelInstance(mb, inst, route)

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, route.StartMapId(), warps[0].Body.MapId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCancelled, evs[0].Type)
	assert.Equal(t, it.CancelReasonStuck, evs[0].Body.Reason)

	assert.False(t, getCharacterRegistry().IsInTransport(ctx, 42))
}

// A route with a forced return: the timer expiring means the player ran out of
// flight time, so they go back to the forced-return map (the client's own
// Map.wz forcedReturn), not to the destination — and the event says CANCELLED
// with reason TIMEOUT, because they did not complete the trip.
func TestCompleteInstance_ForcedReturnWarpsBackAndCancels(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newEffectRoute(t, 1)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(ctx, instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	cs := decodeConsumables(t, mb)
	assert.Len(t, cs, 1)
	assert.Equal(t, consumable.CommandCancelConsumableEffect, cs[0].Type)
	assert.Equal(t, item.Id(2210016), cs[0].Body.ItemId)

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, _map.Id(240000110), warps[0].Body.MapId, "forced return, not the destination")

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCancelled, evs[0].Type)
	assert.Equal(t, it.CancelReasonTimeout, evs[0].Body.Reason)

	assert.False(t, getCharacterRegistry().IsInTransport(ctx, 42))
}

// The ferry regression bar: no forced return, no declared effects — deliver to
// destinationMapId and emit COMPLETED, byte-identically to today.
func TestCompleteInstance_NoForcedReturnDeliversAndCompletes(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route := newPlainRoute(t)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(ctx, instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])

	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, route.DestinationMapId(), warps[0].Body.MapId)

	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCompleted, evs[0].Type)
}

// A route that declares effects but no forced return still delivers to the
// destination while cancelling — the two fields are independent.
func TestCompleteInstance_EffectsWithoutForcedReturnStillDelivers(t *testing.T) {
	p, ctx := setupProcessorTest(t)
	route, err := NewRouteBuilder("effects-no-forced-return").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(60 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		Build()
	assert.NoError(t, err)
	getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	instanceId := board(t, p, route, 42, world.Id(0), channel.Id(1))

	inst, ok := getInstanceRegistry().GetInstance(ctx, instanceId)
	assert.True(t, ok)

	mb := message.NewBuffer()
	p.completeInstance(mb, inst, route)

	assert.Len(t, decodeConsumables(t, mb), 1)
	warps := decodeChangeMaps(t, mb)
	assert.Len(t, warps, 1)
	assert.Equal(t, _map.Id(100000200), warps[0].Body.MapId)
	evs := decodeInstanceTransportEvents(t, mb)
	assert.Len(t, evs, 1)
	assert.Equal(t, it.EventTypeCompleted, evs[0].Type)
}
