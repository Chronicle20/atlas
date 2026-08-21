package position

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func positionTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestRegistry_PutThenLookup(t *testing.T) {
	tn := positionTestTenant(t)

	GetRegistry().Put(tn, 42, Position{X: 100, Y: 200})

	got, ok := GetRegistry().Lookup(tn, 42)
	if !ok {
		t.Fatal("Lookup after Put: want ok == true, got false")
	}
	if got != (Position{X: 100, Y: 200}) {
		t.Errorf("Lookup after Put: want Position{100, 200}, got %+v", got)
	}
}

func TestRegistry_LookupMiss(t *testing.T) {
	tn := positionTestTenant(t)

	got, ok := GetRegistry().Lookup(tn, 99)
	if ok {
		t.Errorf("Lookup with nothing stored: want ok == false, got true (value %+v)", got)
	}
	if got != (Position{}) {
		t.Errorf("Lookup with nothing stored: want zero Position, got %+v", got)
	}
}

func TestRegistry_TenantIsolated(t *testing.T) {
	tnA := positionTestTenant(t)
	tnB := positionTestTenant(t)

	GetRegistry().Put(tnA, 42, Position{X: 1, Y: 2})

	if _, ok := GetRegistry().Lookup(tnB, 42); ok {
		t.Error("Lookup for a different tenant's same character id: want ok == false, got true")
	}
}

func TestRegistry_PutOverwrites(t *testing.T) {
	tn := positionTestTenant(t)

	GetRegistry().Put(tn, 42, Position{X: 1, Y: 2})
	GetRegistry().Put(tn, 42, Position{X: 3, Y: 4})

	got, ok := GetRegistry().Lookup(tn, 42)
	if !ok {
		t.Fatal("Lookup after two Puts: want ok == true, got false")
	}
	if got != (Position{X: 3, Y: 4}) {
		t.Errorf("Lookup after two Puts: want the second value Position{3, 4}, got %+v", got)
	}
}

func TestRegistry_Clear(t *testing.T) {
	tn := positionTestTenant(t)

	GetRegistry().Put(tn, 42, Position{X: 1, Y: 2})
	GetRegistry().Clear(tn, 42)

	if _, ok := GetRegistry().Lookup(tn, 42); ok {
		t.Error("Lookup after Clear: want ok == false, got true")
	}
}
