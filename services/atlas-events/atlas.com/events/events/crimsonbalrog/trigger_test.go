package crimsonbalrog

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/registry"
	"atlas-events/event/scheduling"
	"atlas-events/event/transition"
	transport "atlas-events/kafka/message/transport"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testTenantId is fixed (rather than generated per-test) so seedDefinition,
// testCtx and departedOn/departedAtOn all agree on the same tenant without
// having to thread it through every helper call the brief's test bodies use.
var testTenantId = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

const (
	routeA = "route-a"
	routeB = "route-b"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, definition.MigrateTable, occurrence.MigrateTable, scheduling.MigrateTable, transition.MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(testTenantId)
}

var registerOnce sync.Once

func ensureHandlerRegistered() {
	registerOnce.Do(func() { registry.Register(NewHandler()) })
}

// defOpt customizes a definition seeded by seedDefinition.
type defOpt func(*testDefSpec)

type testDefSpec struct {
	enabled bool
	cfg     Config
}

func enabled(v bool) defOpt { return func(s *testDefSpec) { s.enabled = v } }

func routes(slugs ...string) defOpt {
	return func(s *testDefSpec) { s.cfg.ApplicableRouteIds = slugs }
}

func delay(d time.Duration) defOpt { return func(s *testDefSpec) { s.cfg.TriggerDelay = Duration(d) } }

func jitter(d time.Duration) defOpt {
	return func(s *testDefSpec) { s.cfg.TriggerDelayJitter = Duration(d) }
}

// seedDefinition creates and (optionally) enables a CRIMSON_BALROG definition
// with a valid base configuration, customized by opts.
func seedDefinition(t *testing.T, db *gorm.DB, theType string, opts ...defOpt) definition.Model {
	t.Helper()
	ensureHandlerRegistered()

	spec := testDefSpec{
		cfg: Config{
			AttackProbability: 1,
			MonsterCount:      1,
			AttackMaps: []AttackMap{
				{MapId: 200090010, SpawnPositions: []Position{{X: 0, Y: 0}}},
			},
			Visual: VisualConfig{Name: "balrog"},
		},
	}
	for _, o := range opts {
		o(&spec)
	}

	raw, err := json.Marshal(spec.cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	m, err := definition.NewBuilder(theType, "test-definition").SetConfiguration(raw).Build()
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}

	p := definition.NewProcessor(testLogger(t), testCtx(t), db)
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if spec.enabled {
		created, err = p.SetEnabled(created.Id(), true)
		if err != nil {
			t.Fatalf("SetEnabled: %v", err)
		}
	}

	return created
}

// departedOn builds a VOYAGE_DEPARTED event on the given route slug,
// departing now.
func departedOn(slug string) transport.StatusEvent[transport.VoyageStatusEventBody] {
	return departedAtOn(slug, time.Now().UTC())
}

// departedAtOn builds a VOYAGE_DEPARTED event on the given route slug,
// departing at departedAt. RouteId is tenant-derived from the same slug via
// tenant.DerivedId, exactly as the trigger processor derives it.
func departedAtOn(slug string, departedAt time.Time) transport.StatusEvent[transport.VoyageStatusEventBody] {
	return transport.StatusEvent[transport.VoyageStatusEventBody]{
		RouteId: tenant.DerivedId(testTenantId, "routes", slug),
		Type:    transport.EventStatusVoyageDeparted,
		Body: transport.VoyageStatusEventBody{
			VoyageId:         uuid.New(),
			WorldId:          world.Id(0),
			ChannelId:        channel.Id(1),
			StagingMapId:     _map.Id(200090000),
			EnRouteMapIds:    []_map.Id{200090001, 200090002},
			DestinationMapId: _map.Id(200090010),
			ObservationMapId: _map.Id(200090020),
			DepartedAt:       departedAt,
		},
	}
}

func readAllWork(t *testing.T, db *gorm.DB) []scheduling.Entity {
	t.Helper()
	var out []scheduling.Entity
	if err := db.Order("execute_at asc").Find(&out).Error; err != nil {
		t.Fatalf("readAllWork: %v", err)
	}
	return out
}

// FR-B2: one TRIGGER_EVALUATION per surviving definition, with the full voyage
// scope in context. FR-B3: no occurrence yet.
func TestVoyageDepartedSchedulesOneEvaluationPerApplicableDefinition(t *testing.T) {
	db := newTestDB(t)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA))
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeB))  // different route
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(false), routes(routeA)) // disabled

	if err := NewTriggerProcessor(testLogger(t), testCtx(t), db).OnVoyageDeparted(departedOn(routeA)); err != nil {
		t.Fatalf("OnVoyageDeparted: %v", err)
	}

	work := readAllWork(t, db)
	if len(work) != 1 {
		t.Fatalf("scheduled %d rows, want 1 (route B and the disabled definition must be skipped)", len(work))
	}
	if work[0].Type != scheduling.WorkTypeTriggerEvaluation {
		t.Fatalf("type = %s", work[0].Type)
	}

	var occurrences int64
	db.Model(&occurrence.Entity{}).Count(&occurrences)
	if occurrences != 0 {
		t.Fatalf("departure created %d occurrences, want 0 (FR-B3)", occurrences)
	}
}

// FR-B4: redelivery creates no second row.
func TestVoyageDepartedRedeliveryIsANoOp(t *testing.T) {
	db := newTestDB(t)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA))
	p := NewTriggerProcessor(testLogger(t), testCtx(t), db)

	e := departedOn(routeA)
	if err := p.OnVoyageDeparted(e); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := p.OnVoyageDeparted(e); err != nil {
		t.Fatalf("redelivery must not error: %v", err)
	}
	if got := len(readAllWork(t, db)); got != 1 {
		t.Fatalf("%d work rows after redelivery, want 1", got)
	}
}

// The delay is DURABLE: executeAt is departedAt + delay + a jitter rolled once,
// now, and stored. Restarting the service cannot re-roll it.
func TestExecuteAtIsDeparturePlusDelayPlusRolledJitter(t *testing.T) {
	db := newTestDB(t)
	departedAt := time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA),
		delay(3*time.Minute), jitter(60*time.Second))

	if err := NewTriggerProcessor(testLogger(t), testCtx(t), db).
		OnVoyageDeparted(departedAtOn(routeA, departedAt)); err != nil {
		t.Fatalf("OnVoyageDeparted: %v", err)
	}

	at := readAllWork(t, db)[0].ExecuteAt
	lo := departedAt.Add(3 * time.Minute)
	hi := lo.Add(60 * time.Second)
	if at.Before(lo) || at.After(hi) {
		t.Fatalf("executeAt %s outside [%s, %s]", at, lo, hi)
	}
}

// The whole voyage scope rides on the work row, so evaluation needs no
// follow-up query to learn its maps (FR-V4).
func TestWorkContextCarriesTheFullVoyageScope(t *testing.T) {
	db := newTestDB(t)
	seedDefinition(t, db, "CRIMSON_BALROG", enabled(true), routes(routeA))

	e := departedOn(routeA)
	if err := NewTriggerProcessor(testLogger(t), testCtx(t), db).OnVoyageDeparted(e); err != nil {
		t.Fatalf("OnVoyageDeparted: %v", err)
	}

	work := readAllWork(t, db)
	if len(work) != 1 {
		t.Fatalf("scheduled %d rows, want 1", len(work))
	}

	var wc WorkContext
	if err := json.Unmarshal([]byte(work[0].Context), &wc); err != nil {
		t.Fatalf("unmarshal work context: %v", err)
	}

	if wc.VoyageId != e.Body.VoyageId {
		t.Fatalf("VoyageId = %s, want %s", wc.VoyageId, e.Body.VoyageId)
	}
	if wc.RouteId != e.RouteId {
		t.Fatalf("RouteId = %s, want %s", wc.RouteId, e.RouteId)
	}
	if wc.WorldId != e.Body.WorldId {
		t.Fatalf("WorldId = %v, want %v", wc.WorldId, e.Body.WorldId)
	}
	if wc.ChannelId != e.Body.ChannelId {
		t.Fatalf("ChannelId = %v, want %v", wc.ChannelId, e.Body.ChannelId)
	}
	if wc.StagingMapId != e.Body.StagingMapId {
		t.Fatalf("StagingMapId = %v, want %v", wc.StagingMapId, e.Body.StagingMapId)
	}
	if len(wc.EnRouteMapIds) != len(e.Body.EnRouteMapIds) {
		t.Fatalf("EnRouteMapIds = %v, want %v", wc.EnRouteMapIds, e.Body.EnRouteMapIds)
	}
	for i := range wc.EnRouteMapIds {
		if wc.EnRouteMapIds[i] != e.Body.EnRouteMapIds[i] {
			t.Fatalf("EnRouteMapIds[%d] = %v, want %v", i, wc.EnRouteMapIds[i], e.Body.EnRouteMapIds[i])
		}
	}
	if wc.DestinationMapId != e.Body.DestinationMapId {
		t.Fatalf("DestinationMapId = %v, want %v", wc.DestinationMapId, e.Body.DestinationMapId)
	}
	if wc.ObservationMapId != e.Body.ObservationMapId {
		t.Fatalf("ObservationMapId = %v, want %v", wc.ObservationMapId, e.Body.ObservationMapId)
	}
	if !wc.DepartedAt.Equal(e.Body.DepartedAt) {
		t.Fatalf("DepartedAt = %s, want %s", wc.DepartedAt, e.Body.DepartedAt)
	}
}
