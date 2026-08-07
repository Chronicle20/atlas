package buff

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tn
}

func TestBeaconMirrorSetGetClear(t *testing.T) {
	beaconMirrorOnce = sync.Once{}
	beaconMirror = nil
	m := GetBeaconMirror()
	tn := newTestTenant(t)

	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("empty mirror must miss")
	}

	m.Set(tn, 100, NewBeaconEntry(5211006, 1, 1000001))
	e, ok := m.Get(tn, 100)
	if !ok || e.MobId() != 1000001 || e.SourceId() != 5211006 {
		t.Fatalf("get after set: got %+v ok=%v", e, ok)
	}

	// Re-set replaces (re-cast on another monster).
	m.Set(tn, 100, NewBeaconEntry(5220011, 10, 1000002))
	e, _ = m.Get(tn, 100)
	if e.MobId() != 1000002 || e.SourceId() != 5220011 {
		t.Fatalf("re-set must replace: got %+v", e)
	}

	m.Clear(tn, 100)
	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("get after clear must miss")
	}
}

func TestBeaconMirrorTenantIsolation(t *testing.T) {
	beaconMirrorOnce = sync.Once{}
	beaconMirror = nil
	m := GetBeaconMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)

	m.Set(t1, 100, NewBeaconEntry(5211006, 1, 1000001))
	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant 2 must not see tenant 1's beacon")
	}
}
