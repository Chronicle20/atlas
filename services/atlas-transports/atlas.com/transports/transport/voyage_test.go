package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func voyageTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// FR-V5: the same voyage derives the same id, so a process that restarted
// mid-trip still matches VOYAGE_ARRIVED to VOYAGE_DEPARTED.
func TestVoyageIdIsStableForOneVoyage(t *testing.T) {
	tm := voyageTenant(t)
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	a := VoyageId(tm, routeId, tripId, dep)
	b := VoyageId(tm, routeId, tripId, dep.Add(37*time.Minute)) // same calendar day
	if a != b {
		t.Fatalf("id not stable within a day: %s vs %s", a, b)
	}
}

func TestVoyageIdDiffersAcrossConsecutiveDays(t *testing.T) {
	tm := voyageTenant(t)
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	if VoyageId(tm, routeId, tripId, dep) == VoyageId(tm, routeId, tripId, dep.AddDate(0, 0, 1)) {
		t.Fatalf("consecutive days collided")
	}
}

func TestVoyageIdDiffersAcrossTenantsRoutesAndTrips(t *testing.T) {
	tmA := voyageTenant(t)
	tmB, err := tenant.Create(uuid.MustParse("22222222-2222-2222-2222-222222222222"), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	routeId, tripId := uuid.New(), uuid.New()
	dep := time.Date(2026, 8, 15, 13, 5, 0, 0, time.UTC)

	base := VoyageId(tmA, routeId, tripId, dep)
	if base == VoyageId(tmB, routeId, tripId, dep) {
		t.Fatalf("tenants collided")
	}
	if base == VoyageId(tmA, uuid.New(), tripId, dep) {
		t.Fatalf("routes collided")
	}
	if base == VoyageId(tmA, routeId, uuid.New(), dep) {
		t.Fatalf("trips collided")
	}
}
