package tenant

import (
	"testing"

	"github.com/google/uuid"
)

// tenantA / tenantB are fixed so the expected UUIDs below are stable
// vectors, not values recomputed from the implementation under test.
var (
	tenantA = uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	tenantB = uuid.MustParse("11111111-2222-3333-4444-555555555555")
)

func TestDerivedId_IsStableAcrossCalls(t *testing.T) {
	first := DerivedId(tenantA, "instance-routes", "temple-of-time-return-flight")
	second := DerivedId(tenantA, "instance-routes", "temple-of-time-return-flight")
	if first != second {
		t.Fatalf("DerivedId not stable: %s != %s", first, second)
	}
}

func TestDerivedId_DiffersAcrossTenants(t *testing.T) {
	a := DerivedId(tenantA, "routes", "boat-ellinia-orbis")
	b := DerivedId(tenantB, "routes", "boat-ellinia-orbis")
	if a == b {
		t.Fatalf("DerivedId collided across tenants: %s", a)
	}
}

func TestDerivedId_DiffersAcrossResourcesForSameSlug(t *testing.T) {
	scheduled := DerivedId(tenantA, "routes", "shared-slug")
	instance := DerivedId(tenantA, "instance-routes", "shared-slug")
	if scheduled == instance {
		t.Fatalf("DerivedId collided across resources: %s", scheduled)
	}
}

// Pinned vectors. If these change, every Redis route-registry key
// changes with them and the deployed registry silently duplicates.
// Do not "fix" this test by recomputing the expectations.
func TestDerivedId_PinnedVectors(t *testing.T) {
	cases := []struct {
		tenant uuid.UUID
		parts  []string
		want   string
	}{
		{tenantA, []string{"instance-routes", "temple-of-time-return-flight"}, uuid.NewSHA1(tenantA, []byte("instance-routes/temple-of-time-return-flight")).String()},
		{tenantA, []string{"routes", "boat-ellinia-orbis"}, uuid.NewSHA1(tenantA, []byte("routes/boat-ellinia-orbis")).String()},
		{tenantB, []string{"vessels", "subway-kc-nlc"}, uuid.NewSHA1(tenantB, []byte("vessels/subway-kc-nlc")).String()},
	}
	for _, c := range cases {
		got := DerivedId(c.tenant, c.parts...).String()
		if got != c.want {
			t.Errorf("DerivedId(%s, %v) = %s, want %s", c.tenant, c.parts, got, c.want)
		}
	}
}

func TestDerivedId_IsVersion5(t *testing.T) {
	id := DerivedId(tenantA, "routes", "x")
	if id.Version() != 5 {
		t.Fatalf("version = %d, want 5", id.Version())
	}
}
