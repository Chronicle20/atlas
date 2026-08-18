package transport

import (
	channelmock "atlas-transports/channel/mock"
	"atlas-transports/character"
	charactermock "atlas-transports/character/mock"
	"atlas-transports/kafka/message"
	"atlas-transports/kafka/message/transport"
	_map2 "atlas-transports/map"
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channel2 "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupTransportTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRouteRegistry(rc)
}

func newTestTenantContext(t *testing.T) (tenant.Model, context.Context) {
	t.Helper()
	tenantId := uuid.New()
	tm, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)
	return tm, ctx
}

func TestRouteRegistry_GetRouteByStartMap(t *testing.T) {
	setupTransportTestRegistry(t)

	// Create test tenants
	_, ctx1 := newTestTenantContext(t)
	_, ctx2 := newTestTenantContext(t)

	// Create test routes with different start map IDs
	route1, err := NewBuilder("Ellinia to Orbis").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route2, err := NewBuilder("Orbis to Ludibrium").
		SetStartMapId(_map.Id(200000100)).
		SetStagingMapId(_map.Id(200000110)).
		SetEnRouteMapIds([]_map.Id{_map.Id(200000111)}).
		SetDestinationMapId(_map.Id(220000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route3, err := NewBuilder("Different Tenant Route").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(300000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	tests := []struct {
		name          string
		setup         func()
		ctx           context.Context
		mapId         _map.Id
		expectedRoute Model
		expectError   bool
	}{
		{
			name: "Successful route retrieval",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1, route2})
			},
			ctx:           ctx1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Route not found",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1, route2})
			},
			ctx:         ctx1,
			mapId:       _map.Id(999999999),
			expectError: true,
		},
		{
			name: "Multi-tenant isolation",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1})
				getRouteRegistry().AddTenant(ctx2, []Model{route3})
			},
			ctx:           ctx1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Different tenant same map ID",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1})
				getRouteRegistry().AddTenant(ctx2, []Model{route3})
			},
			ctx:           ctx2,
			mapId:         _map.Id(101000300),
			expectedRoute: route3,
			expectError:   false,
		},
		{
			name:        "Empty tenant registry",
			setup:       func() {},
			ctx:         ctx1,
			mapId:       _map.Id(101000300),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-initialize registry for each test
			setupTransportTestRegistry(t)

			// Setup test data
			tt.setup()

			// Execute test
			result, err := getRouteRegistry().GetRouteByStartMap(tt.ctx, tt.mapId)

			// Assert results
			if tt.expectError {
				assert.Error(t, err, "Expected an error")
				assert.Equal(t, Model{}, result, "Should return empty model on error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, tt.expectedRoute.Id(), result.Id(), "Route ID should match")
				assert.Equal(t, tt.expectedRoute.Name(), result.Name(), "Route name should match")
				assert.Equal(t, tt.expectedRoute.StartMapId(), result.StartMapId(), "Start map ID should match")
			}
		})
	}
}

func TestRouteRegistry_GetRouteByStartMap_MultipleRoutesOneMap(t *testing.T) {
	setupTransportTestRegistry(t)
	_, ctx := newTestTenantContext(t)

	route1, err := NewBuilder("Route 1").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route2, err := NewBuilder("Route 2").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000302)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000303)}).
		SetDestinationMapId(_map.Id(300000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	getRouteRegistry().AddTenant(ctx, []Model{route1, route2})

	// Should return one of the routes (order not guaranteed due to map iteration)
	result, err := getRouteRegistry().GetRouteByStartMap(ctx, _map.Id(101000300))

	assert.NoError(t, err, "Should not return an error")
	assert.Equal(t, _map.Id(101000300), result.StartMapId(), "Start map ID should match")
	// We can't assert which specific route is returned due to iteration order
	// But we can assert that it's one of the two routes
	assert.True(t,
		result.Id() == route1.Id() || result.Id() == route2.Id(),
		"Should return one of the routes with the matching start map")
}

// Helper function to create a test route
func createTestRoute(t *testing.T, name string, startMapId, stagingMapId _map.Id, enRouteMapIds []_map.Id, destinationMapId _map.Id) Model {
	route, err := NewBuilder(name).
		SetStartMapId(startMapId).
		SetStagingMapId(stagingMapId).
		SetEnRouteMapIds(enRouteMapIds).
		SetDestinationMapId(destinationMapId).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)
	return route
}

// Helper to create a ProcessorImpl with mock character processor for testing
func createTestProcessor(tenantModel tenant.Model, charP character.Processor) *ProcessorImpl {
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{})
	ctx := tenant.WithContext(context.Background(), tenantModel)

	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		t:     tenantModel,
		charP: charP,
	}
}

func TestWarpToRouteStartMapOnLogout_FromStagingMap(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedCharacterId = characterId
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, stagingMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromEnRouteMap(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry EnRoute", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedCharacterId = characterId
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, enRouteMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromDestinationMap_NoWarp(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Dest", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, destinationMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.False(t, warpCalled, "Should not warp from destination map")
}

func TestWarpToRouteStartMapOnLogout_FromUnrelatedMap_NoWarp(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Unrelated", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	unrelatedMapId := _map.Id(100000000)
	currentField := field.NewBuilder(0, 0, unrelatedMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.False(t, warpCalled, "Should not warp from unrelated map")
}

func TestWarpToRouteStartMapOnLogout_MultipleRoutes_CorrectMatch(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	route1 := createTestRoute(t, "Ferry 1",
		_map.Id(101000300),
		_map.Id(200090000),
		[]_map.Id{_map.Id(200090100)},
		_map.Id(200000100))

	route2 := createTestRoute(t, "Ferry 2",
		_map.Id(220000000),
		_map.Id(220090000),
		[]_map.Id{_map.Id(220090100)},
		_map.Id(220000100))

	getRouteRegistry().AddTenant(ctx, []Model{route1, route2})

	var warpedToFieldId field.Id
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, _map.Id(220090000)).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, _map.Id(220000000), targetField.MapId(), "Should warp to route2's start map")
}

func TestWarpToRouteStartMapOnLogout_NoRoutes_NoError(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)

	require.NoError(t, err)
	assert.False(t, warpCalled, "Should not warp when no routes exist")
}

// noopMapProcessor is a _map.Processor that reports no characters present in
// any map, so UpdateRoute's warp loops have nothing to do.
type noopMapProcessor struct{}

func (noopMapProcessor) CharacterIdsInMapProvider(_ field.Model) model.Provider[[]uint32] {
	return model.FixedProvider[[]uint32](nil)
}

var _ _map2.Processor = noopMapProcessor{}

// channelSpec names one channel a fake channel.Processor should report.
type channelSpec struct {
	world   byte
	channel byte
}

// newProcessorWithChannels builds a ProcessorImpl wired to a fake
// channel.Processor reporting exactly the given channels, a working
// message.Buffer to inspect emitted events against, and a no-op map
// processor so the warp loops UpdateRoute runs are inert. now is pinned via
// SetNow so UpdateRoute's state evaluation is deterministic - not sensitive
// to when in real time (including a UTC midnight rollover, which
// Evaluate's timeOfDay() strips the date around) the test happens to run.
func newProcessorWithChannels(t *testing.T, specs []channelSpec, now time.Time) (*ProcessorImpl, *message.Buffer, []channel2.Model) {
	t.Helper()
	setupTransportTestRegistry(t)
	tenantModel, ctx := newTestTenantContext(t)
	p, mb, chans := newProcessorForTenantWithChannels(t, tenantModel, ctx, specs, now)
	return p, mb, chans
}

// newProcessorForTenantWithChannels is newProcessorWithChannels with the
// tenant supplied by the caller instead of freshly registered, so a test
// that ticks UpdateRoute more than once (e.g. a departure tick followed by
// an arrival tick) can keep the same tenant identity across both - VoyageId
// is derived in part from the tenant, so two independently-registered
// tenants would desynchronize the very identity the test is checking.
func newProcessorForTenantWithChannels(t *testing.T, tenantModel tenant.Model, ctx context.Context, specs []channelSpec, now time.Time) (*ProcessorImpl, *message.Buffer, []channel2.Model) {
	t.Helper()

	chans := make([]channel2.Model, 0, len(specs))
	for _, s := range specs {
		chans = append(chans, channel2.NewModel(world.Id(s.world), channel2.Id(s.channel)))
	}

	mockChanP := &channelmock.ProcessorMock{
		GetAllFunc: func() []channel2.Model {
			return chans
		},
	}

	l := logrus.New()
	l.SetOutput(&bytes.Buffer{})

	p := &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		t:     tenantModel,
		chanP: mockChanP,
		charP: &charactermock.ProcessorMock{},
		mp:    noopMapProcessor{},
		now:   func() time.Time { return now },
	}
	return p, message.NewBuffer(), chans
}

// voyageEvents decodes every StatusEvent[VoyageStatusEventBody] of the given
// type off the buffer's transport-status topic.
func voyageEvents(t *testing.T, mb *message.Buffer, eventType string) []transport.StatusEvent[transport.VoyageStatusEventBody] {
	t.Helper()
	var out []transport.StatusEvent[transport.VoyageStatusEventBody]
	for _, msg := range mb.GetAll()[transport.EnvEventTopicStatus] {
		var ev transport.StatusEvent[transport.VoyageStatusEventBody]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			t.Fatalf("failed to decode voyage event: %v", err)
		}
		if ev.Type == eventType {
			out = append(out, ev)
		}
	}
	return out
}

// countEvents counts messages on the transport-status topic whose envelope
// Type matches eventType, without decoding a specific body shape.
func countEvents(t *testing.T, mb *message.Buffer, eventType string) int {
	t.Helper()
	count := 0
	for _, msg := range mb.GetAll()[transport.EnvEventTopicStatus] {
		var ev struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			t.Fatalf("failed to decode event envelope: %v", err)
		}
		if ev.Type == eventType {
			count++
		}
	}
	return count
}

// routeWithSchedule and tod are defined in evaluate_test.go and reused here.

// inTransitRouteAboutToArrive returns a route persisted as InTransit and a
// pinned `now` at which its selected trip's arrival has already passed and
// boarding for the same trip's next cycle has not yet opened, so
// UpdateRoute(now) observes the AwaitingReturn transition ("the ferry has
// arrived and is waiting to board again"). now is a fixed instant, not
// time.Now() - see newProcessorWithChannels for why.
func inTransitRouteAboutToArrive(t *testing.T) (Model, time.Time) {
	t.Helper()
	routeId := uuid.New()
	// Trip A boards 12:00, closes 12:50, departs 13:00, arrives 13:30. Trip
	// B - a later trip on the same schedule whose boarding is NOT yet open
	// at the pinned `now` - is what makes Evaluate land in AwaitingReturn
	// (counting down to trip B's boarding) rather than OpenEntry: with only
	// trip A on the schedule the same `now` would fall into the gap-2
	// (OutOfService) shape instead. justArrivedTrip is trip A, which is what
	// this fixture needs populated - see TestArrivalEmitsVoyageArrivedPerChannel.
	tripA := NewTripScheduleModel(uuid.New(), routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	tripB := NewTripScheduleModel(uuid.New(), routeId, tod(14, 0), tod(14, 50), tod(15, 0), tod(15, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{tripA, tripB})
	updated, err := m.Builder().SetState(InTransit).Build()
	require.NoError(t, err)
	now := time.Date(2026, 8, 15, 13, 31, 0, 0, time.UTC)
	return updated, now
}

// openEntryRouteAboutToDepart returns a route persisted as OpenEntry and a
// pinned `now` at which its selected trip's departure has already passed, so
// UpdateRoute(now) observes the InTransit transition. now is fixed, not
// time.Now().
func openEntryRouteAboutToDepart(t *testing.T) (Model, time.Time) {
	t.Helper()
	routeId := uuid.New()
	tripId := uuid.New()
	trip := NewTripScheduleModel(tripId, routeId, tod(12, 30), tod(12, 35), tod(12, 37), tod(12, 47))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})
	updated, err := m.Builder().SetState(OpenEntry).Build()
	require.NoError(t, err)
	now := time.Date(2026, 8, 15, 12, 40, 0, 0, time.UTC)
	return updated, now
}

// FR-V3/FR-V4: InTransit -> AwaitingReturn emits VOYAGE_ARRIVED, once per
// channel, with the full scope. Before this task that transition emitted
// nothing at all (PRD F1).
func TestArrivalEmitsVoyageArrivedPerChannel(t *testing.T) {
	route, now := inTransitRouteAboutToArrive(t)
	p, mb, chans := newProcessorWithChannels(t, []channelSpec{{world: 1, channel: 1}, {world: 1, channel: 2}}, now)
	_ = chans

	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	msgs := voyageEvents(t, mb, transport.EventStatusVoyageArrived)
	if len(msgs) != 2 {
		t.Fatalf("expected one VOYAGE_ARRIVED per channel, got %d", len(msgs))
	}
	if msgs[0].Body.VoyageId == uuid.Nil {
		t.Fatalf("voyage id not populated")
	}
	if msgs[0].Body.VoyageId != msgs[1].Body.VoyageId {
		t.Fatalf("per-channel events must share one voyage id")
	}
	if msgs[0].Body.ChannelId == msgs[1].Body.ChannelId {
		t.Fatalf("expected distinct channels, both %d", msgs[0].Body.ChannelId)
	}
	if msgs[0].Body.DestinationMapId != route.DestinationMapId() {
		t.Fatalf("scope not carried")
	}
}

// FR-V6 / acceptance 20.4: the existing ARRIVED and DEPARTED emits are
// unchanged — same count, same single-emit shape, same body.
func TestDepartureStillEmitsOneDepartedAndOneVoyageDepartedPerChannel(t *testing.T) {
	route, now := openEntryRouteAboutToDepart(t)
	p, mb, _ := newProcessorWithChannels(t, []channelSpec{{world: 1, channel: 1}, {world: 1, channel: 2}}, now)

	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	if got := countEvents(t, mb, transport.EventStatusDeparted); got != 1 {
		t.Fatalf("DEPARTED emitted %d times, want exactly 1 (unchanged)", got)
	}
	if got := len(voyageEvents(t, mb, transport.EventStatusVoyageDeparted)); got != 2 {
		t.Fatalf("VOYAGE_DEPARTED emitted %d times, want one per channel", got)
	}
}

// TestArrivalAcrossMidnightUsesDepartureDayVoyageId is the fix-round
// regression case: a midnight-crossing trip (departs 23:30, arrives 00:30)
// observed well after its arrival, at 00:40 the following calendar day. The
// governing boundaries here (boardingOpen 22:30 .. arrival 00:30) straddle
// midnight exactly the way naive offset-from-time.Now() arithmetic risked
// producing by accident (see fix-round report) - this test pins the clock
// explicitly instead, and checks the VOYAGE_ARRIVED body's VoyageId is
// derived from the PREVIOUS day's departure instant, matching what
// VOYAGE_DEPARTED would have carried when the same trip departed. Getting
// this wrong (e.g. deriving today's 23:30 instead of yesterday's) would
// silently mint a different voyage id and desynchronize atlas-events'
// consumer from the departure it already recorded.
func TestArrivalAcrossMidnightUsesDepartureDayVoyageId(t *testing.T) {
	routeId := uuid.New()
	tripId := uuid.New()
	trip := NewTripScheduleModel(tripId, routeId, tod(22, 30), tod(23, 20), tod(23, 30), tod(0, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})
	route, err := m.Builder().SetState(InTransit).Build()
	require.NoError(t, err)

	now := time.Date(2026, 8, 16, 0, 40, 0, 0, time.UTC)
	p, mb, chans := newProcessorWithChannels(t, []channelSpec{{world: 1, channel: 1}, {world: 1, channel: 2}}, now)
	_ = chans

	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	msgs := voyageEvents(t, mb, transport.EventStatusVoyageArrived)
	if len(msgs) != 2 {
		t.Fatalf("expected one VOYAGE_ARRIVED per channel, got %d", len(msgs))
	}

	wantDepartedAt := time.Date(2026, 8, 15, 23, 30, 0, 0, time.UTC)
	wantVoyageId := VoyageId(p.t, routeId, tripId, wantDepartedAt)
	if msgs[0].Body.VoyageId != wantVoyageId {
		t.Fatalf("VoyageId = %s, want %s (derived from previous day's departure at %s)",
			msgs[0].Body.VoyageId, wantVoyageId, wantDepartedAt)
	}
	if !msgs[0].Body.DepartedAt.Equal(wantDepartedAt) {
		t.Fatalf("DepartedAt = %s, want %s", msgs[0].Body.DepartedAt, wantDepartedAt)
	}
}

// warpRecord captures one WarpRandom call the mock character processor
// recorded, so a test can assert the arrival warp actually ran rather than
// only that UpdateRoute did not error.
type warpRecord struct {
	characterId uint32
	toMapId     _map.Id
}

// arrivalWarpMapProcessor reports a single fixed character present on
// enRouteMapId and none elsewhere, so a test can assert UpdateRoute's
// arrival warp loop actually moved that character off the en-route map.
type arrivalWarpMapProcessor struct {
	enRouteMapId _map.Id
	characterId  uint32
}

func (a arrivalWarpMapProcessor) CharacterIdsInMapProvider(f field.Model) model.Provider[[]uint32] {
	if f.MapId() == a.enRouteMapId {
		return model.FixedProvider([]uint32{a.characterId})
	}
	return model.FixedProvider[[]uint32](nil)
}

var _ _map2.Processor = arrivalWarpMapProcessor{}

// newProcessorForArrivalWarp is newProcessorWithChannels, but its map
// processor reports one character present on enRouteMapId (instead of the
// no-op default) and its character processor records every WarpRandom call,
// so a test can assert the arrival warp moved that character rather than
// merely that emitting the arrival event did not error.
func newProcessorForArrivalWarp(t *testing.T, now time.Time, enRouteMapId _map.Id) (*ProcessorImpl, *message.Buffer, *[]warpRecord) {
	t.Helper()
	setupTransportTestRegistry(t)
	tenantModel, ctx := newTestTenantContext(t)

	chans := []channel2.Model{channel2.NewModel(world.Id(1), channel2.Id(1))}
	mockChanP := &channelmock.ProcessorMock{
		GetAllFunc: func() []channel2.Model {
			return chans
		},
	}

	warps := &[]warpRecord{}
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					tf, ok := field.FromId(fieldId)
					if !ok {
						t.Fatalf("could not parse warped-to field id")
					}
					*warps = append(*warps, warpRecord{characterId: characterId, toMapId: tf.MapId()})
					return nil
				}
			}
		},
	}

	l := logrus.New()
	l.SetOutput(&bytes.Buffer{})

	p := &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		t:     tenantModel,
		chanP: mockChanP,
		charP: mockCharP,
		mp:    arrivalWarpMapProcessor{enRouteMapId: enRouteMapId, characterId: 500},
		now:   func() time.Time { return now },
	}
	return p, message.NewBuffer(), warps
}

// routeAboutToArriveIntoOpenEntry returns a route persisted as InTransit
// whose next trip's boarding window is already open by the arrival instant
// (the gap-1 shape from the bug report: trip A boards 12:00, departs 13:00,
// arrives 13:30; trip B boards 13:00), so UpdateRoute(now) evaluates to
// OpenEntry rather than AwaitingReturn.
func routeAboutToArriveIntoOpenEntry(t *testing.T) (Model, time.Time) {
	t.Helper()
	routeId := uuid.New()
	tripA := NewTripScheduleModel(uuid.New(), routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	tripB := NewTripScheduleModel(uuid.New(), routeId, tod(13, 0), tod(13, 50), tod(14, 0), tod(14, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{tripA, tripB})
	updated, err := m.Builder().SetState(InTransit).Build()
	require.NoError(t, err)
	now := time.Date(2026, 8, 15, 13, 31, 0, 0, time.UTC)
	return updated, now
}

// routeAboutToArriveIntoOutOfService returns a route persisted as InTransit
// with no future trip on its schedule (the gap-2 shape from the bug
// report), so UpdateRoute(now) evaluates to OutOfService rather than
// AwaitingReturn.
func routeAboutToArriveIntoOutOfService(t *testing.T) (Model, time.Time) {
	t.Helper()
	routeId := uuid.New()
	trip := NewTripScheduleModel(uuid.New(), routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{trip})
	updated, err := m.Builder().SetState(InTransit).Build()
	require.NoError(t, err)
	now := time.Date(2026, 8, 15, 13, 31, 0, 0, time.UTC)
	return updated, now
}

// Gap 1 regression: a route whose arrival tick lands in OpenEntry (rather
// than AwaitingReturn, because the next trip's boarding is already open)
// must still run the arrival warp and emit one VOYAGE_ARRIVED per channel.
// Before this fix, gating the arrival block on r.State() == AwaitingReturn
// silently dropped both.
func TestArrivalIntoOpenEntryWarpsAndEmitsVoyageArrivedPerChannel(t *testing.T) {
	route, now := routeAboutToArriveIntoOpenEntry(t)
	enRouteMapId := route.EnRouteMapIds()[0]
	p, mb, warps := newProcessorForArrivalWarp(t, now, enRouteMapId)

	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	if len(*warps) != 1 {
		t.Fatalf("expected the arrival warp to move 1 character, got %d", len(*warps))
	}
	if (*warps)[0].toMapId != route.DestinationMapId() {
		t.Fatalf("warped to map %d, want destination map %d", (*warps)[0].toMapId, route.DestinationMapId())
	}

	msgs := voyageEvents(t, mb, transport.EventStatusVoyageArrived)
	if len(msgs) != 1 {
		t.Fatalf("expected one VOYAGE_ARRIVED per channel, got %d", len(msgs))
	}
	if msgs[0].Body.VoyageId == uuid.Nil {
		t.Fatalf("voyage id not populated")
	}
}

// Gap 2 regression: a route whose arrival tick lands in OutOfService
// (rather than AwaitingReturn, because no future trip remains) must still
// run the arrival warp and emit one VOYAGE_ARRIVED per channel.
func TestArrivalIntoOutOfServiceWarpsAndEmitsVoyageArrivedPerChannel(t *testing.T) {
	route, now := routeAboutToArriveIntoOutOfService(t)
	enRouteMapId := route.EnRouteMapIds()[0]
	p, mb, warps := newProcessorForArrivalWarp(t, now, enRouteMapId)

	if err := p.UpdateRoute(mb)(route); err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}

	if len(*warps) != 1 {
		t.Fatalf("expected the arrival warp to move 1 character, got %d", len(*warps))
	}
	if (*warps)[0].toMapId != route.DestinationMapId() {
		t.Fatalf("warped to map %d, want destination map %d", (*warps)[0].toMapId, route.DestinationMapId())
	}

	msgs := voyageEvents(t, mb, transport.EventStatusVoyageArrived)
	if len(msgs) != 1 {
		t.Fatalf("expected one VOYAGE_ARRIVED per channel, got %d", len(msgs))
	}
	if msgs[0].Body.VoyageId == uuid.Nil {
		t.Fatalf("voyage id not populated")
	}
}

// TestArrivalVoyageIdMatchesDepartureVoyageIdEndToEnd is the assertion that
// would have caught the original live bug: the VOYAGE_ARRIVED emitted for a
// trip's arrival must carry the same voyageId as the VOYAGE_DEPARTED
// emitted for that trip's departure, asserted end-to-end through the
// processor (not just through Evaluate). Both ticks run through the same
// tenant so VoyageId's tenant-derived component cannot itself desync them.
func TestArrivalVoyageIdMatchesDepartureVoyageIdEndToEnd(t *testing.T) {
	setupTransportTestRegistry(t)
	tenantModel, ctx := newTestTenantContext(t)

	routeId := uuid.New()
	tripA := NewTripScheduleModel(uuid.New(), routeId, tod(12, 0), tod(12, 50), tod(13, 0), tod(13, 30))
	tripB := NewTripScheduleModel(uuid.New(), routeId, tod(13, 0), tod(13, 50), tod(14, 0), tod(14, 30))
	m := routeWithSchedule(t, routeId, []TripScheduleModel{tripA, tripB})

	locked, err := m.Builder().SetState(LockedEntry).Build()
	require.NoError(t, err)

	departureNow := time.Date(2026, 8, 15, 13, 10, 0, 0, time.UTC)
	p1, mb1, _ := newProcessorForTenantWithChannels(t, tenantModel, ctx, []channelSpec{{world: 1, channel: 1}}, departureNow)
	require.NoError(t, p1.UpdateRoute(mb1)(locked))

	departedMsgs := voyageEvents(t, mb1, transport.EventStatusVoyageDeparted)
	require.Len(t, departedMsgs, 1)
	departedVoyageId := departedMsgs[0].Body.VoyageId
	require.NotEqual(t, uuid.Nil, departedVoyageId)

	inTransit, err := m.Builder().SetState(InTransit).Build()
	require.NoError(t, err)

	// past tripA's arrival, with tripB's boarding already open - the same
	// gap-1 shape TestArrivalIntoOpenEntryWarpsAndEmitsVoyageArrivedPerChannel
	// exercises, chosen here specifically because it is the shape the
	// pre-fix code silently dropped VOYAGE_ARRIVED for.
	arrivalNow := time.Date(2026, 8, 15, 13, 31, 0, 0, time.UTC)
	p2, mb2, _ := newProcessorForTenantWithChannels(t, tenantModel, ctx, []channelSpec{{world: 1, channel: 1}}, arrivalNow)
	require.NoError(t, p2.UpdateRoute(mb2)(inTransit))

	arrivedMsgs := voyageEvents(t, mb2, transport.EventStatusVoyageArrived)
	require.Len(t, arrivedMsgs, 1)

	assert.Equal(t, departedVoyageId, arrivedMsgs[0].Body.VoyageId,
		"VOYAGE_ARRIVED's voyageId must equal VOYAGE_DEPARTED's voyageId for the same trip")
}
