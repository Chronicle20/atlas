package config_test

import (
	"atlas-transports/transport/config"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTenant(t *testing.T, id uuid.UUID) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func sampleRoute(id, uuidAttr string) config.RouteRestModel {
	return config.RouteRestModel{
		Id:                     id,
		Uuid:                   uuidAttr,
		Name:                   id,
		StartMapId:             540010000,
		StagingMapId:           540010001,
		EnRouteMapIds:          []_map.Id{540010002},
		DestinationMapId:       103000000,
		ObservationMapId:       540010000,
		BoardingWindowDuration: time.Duration(4),
		PreDepartureDuration:   time.Duration(1),
		TravelDuration:         time.Duration(1),
		CycleInterval:          time.Duration(6),
	}
}

func TestExtractRouteFor_UsesSuppliedUuid(t *testing.T) {
	tm := testTenant(t, uuid.New())
	want := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleRoute("boat-ellinia-orbis", want.String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if m.Id() != want {
		t.Fatalf("id = %s, want %s", m.Id(), want)
	}
}

func TestExtractRouteFor_IsStableAcrossRepeatedCalls(t *testing.T) {
	extract := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))
	first, err := extract(sampleRoute("boat-ellinia-orbis", ""))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := extract(sampleRoute("boat-ellinia-orbis", ""))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.Id() != second.Id() {
		t.Fatalf("ids differ across calls: %s != %s", first.Id(), second.Id())
	}
}

func TestExtractRouteFor_DiffersAcrossTenantsForSameSlug(t *testing.T) {
	a, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	b, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant B: %v", err)
	}
	if a.Id() == b.Id() {
		t.Fatalf("ids collided across tenants: %s", a.Id())
	}
}

func TestExtractRouteFor_FallbackMatchesTheSharedFormula(t *testing.T) {
	id := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), testTenant(t, id))(sampleRoute("boat-ellinia-orbis", "not-a-uuid"))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	want := tenant.DerivedId(id, "routes", "boat-ellinia-orbis")
	if m.Id() != want {
		t.Fatalf("id = %s, want the derived %s", m.Id(), want)
	}
}

// ExtractVessel is deliberately untouched — it already sets a stable
// slug id, which is why vessels never drifted.
func TestExtractVessel_StillUsesSlugId(t *testing.T) {
	v, err := config.ExtractVessel(config.VesselRestModel{
		Id:              "boat-ellinia-orbis",
		Name:            "boat-ellinia-orbis",
		RouteAID:        "boat-ellinia-orbis",
		RouteBID:        "boat-orbis-ellinia",
		TurnaroundDelay: time.Duration(5),
	})
	if err != nil {
		t.Fatalf("ExtractVessel: %v", err)
	}
	if v.Id() != "boat-ellinia-orbis" {
		t.Fatalf("vessel id = %q, want the slug", v.Id())
	}
}
