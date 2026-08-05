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
	getCharacterRegistry().Add(42, uuid.New())

	mb := message.NewBuffer()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(240000110)).Build()
	assert.Error(t, p.StartTransport(mb)(42, route.Id(), f))

	assert.Empty(t, mb.GetAll()[consumable.EnvCommandTopic])
}
