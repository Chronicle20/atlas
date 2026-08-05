package config_test

import (
	"atlas-transports/instance/config"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// If processor_drain_test.go in this package already declares a helper
// with one of these names, reuse it instead of redeclaring — both files
// are package config_test and would collide.
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

func sampleInstanceRoute(id, uuidAttr string) config.InstanceRouteRestModel {
	return config.InstanceRouteRestModel{
		Id:                    id,
		Uuid:                  uuidAttr,
		Name:                  id,
		StartMapId:            270000100,
		TransitMapIds:         []_map.Id{200090510},
		DestinationMapId:      240000110,
		Capacity:              1,
		BoardingWindowSeconds: 1,
		TravelDurationSeconds: 900,
	}
}

func TestExtractRouteFor_UsesSuppliedUuid(t *testing.T) {
	tm := testTenant(t, uuid.New())
	want := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("temple-of-time-return-flight", want.String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if m.Id() != want {
		t.Fatalf("id = %s, want %s (the supplied uuid attribute)", m.Id(), want)
	}
}

func TestExtractRouteFor_IsStableAcrossRepeatedCalls(t *testing.T) {
	tm := testTenant(t, uuid.New())
	extract := config.ExtractRouteFor(quietLogger(), tm)
	first, err := extract(sampleInstanceRoute("temple-of-time-return-flight", ""))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := extract(sampleInstanceRoute("temple-of-time-return-flight", ""))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.Id() != second.Id() {
		t.Fatalf("ids differ across calls: %s != %s", first.Id(), second.Id())
	}
}

func TestExtractRouteFor_DiffersAcrossTenantsForSameSlug(t *testing.T) {
	a, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleInstanceRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	b, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleInstanceRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant B: %v", err)
	}
	if a.Id() == b.Id() {
		t.Fatalf("ids collided across tenants: %s", a.Id())
	}
}

func TestExtractRouteFor_FallbackMatchesTheSharedFormula(t *testing.T) {
	id := uuid.New()
	tm := testTenant(t, id)
	for _, raw := range []string{"", "not-a-uuid"} {
		m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("temple-of-time-return-flight", raw))
		if err != nil {
			t.Fatalf("raw=%q: %v", raw, err)
		}
		want := tenant.DerivedId(id, "instance-routes", "temple-of-time-return-flight")
		if m.Id() != want {
			t.Fatalf("raw=%q: id = %s, want the derived %s", raw, m.Id(), want)
		}
	}
}

// sampleInstanceRoute deliberately omits the effect attributes, so this test
// builds its own fixture: ExtractRouteFor must carry both through to the
// domain model or the processor sees zero values with no error anywhere.
func TestExtractRouteFor_ThreadsEffectAttributes(t *testing.T) {
	tm := testTenant(t, uuid.New())
	r := sampleInstanceRoute("temple-of-time-flight", uuid.New().String())
	r.EffectItemIds = []item.Id{2210016}
	r.ForcedReturnMapId = _map.Id(240000110)

	m, err := config.ExtractRouteFor(quietLogger(), tm)(r)
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if got := m.EffectItemIds(); len(got) != 1 || got[0] != item.Id(2210016) {
		t.Fatalf("EffectItemIds = %v, want [2210016]", got)
	}
	if m.ForcedReturnMapId() != _map.Id(240000110) {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", m.ForcedReturnMapId())
	}
}

// A route declaring neither attribute must still build — the ten unaffected
// routes take this path.
func TestExtractRouteFor_EffectAttributesAreOptional(t *testing.T) {
	tm := testTenant(t, uuid.New())
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("ellinia-ereve-ferry", uuid.New().String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if len(m.EffectItemIds()) != 0 {
		t.Fatalf("EffectItemIds = %v, want empty", m.EffectItemIds())
	}
	if m.ForcedReturnMapId() != _map.Id(0) {
		t.Fatalf("ForcedReturnMapId = %d, want 0", m.ForcedReturnMapId())
	}
}
