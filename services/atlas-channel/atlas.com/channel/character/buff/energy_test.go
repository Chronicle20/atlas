package buff

import (
	"sync"
	"testing"
)

func TestEnergyMirrorSetGetClear(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	tn := newTestTenant(t)

	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("empty mirror must miss")
	}

	m.Set(tn, 100, 4998)
	v, ok := m.Get(tn, 100)
	if !ok || v != 4998 {
		t.Fatalf("get after set: got (%d,%v) want (4998,true)", v, ok)
	}

	// Re-set replaces (each gain overwrites the bar reading).
	m.Set(tn, 100, 15000)
	v, _ = m.Get(tn, 100)
	if v != 15000 {
		t.Fatalf("re-set must replace: got %d want 15000", v)
	}

	m.Clear(tn, 100)
	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("get after clear must miss")
	}
}

// A zero bar is a real reading, not an absence: Get must report ok=true so the
// cast gate rejects rather than failing open.
func TestEnergyMirrorZeroIsPresent(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	tn := newTestTenant(t)

	m.Set(tn, 100, 0)
	v, ok := m.Get(tn, 100)
	if !ok || v != 0 {
		t.Fatalf("zero must be a present reading: got (%d,%v)", v, ok)
	}
}

func TestEnergyMirrorTenantIsolation(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)

	m.Set(t1, 100, 4998)
	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant 2 must not see tenant 1's bar")
	}
}
