package canonical

import (
	"testing"

	"github.com/google/uuid"
)

// frozenGMS83dot1 is the frozen canonical tenant id for GMS 83.1.
// It was computed by running TenantId("GMS", 83, 1) once and pinned here so
// that any future change to Namespace or the format string fails loudly.
const frozenGMS83dot1 = "144ba144-0b45-5635-a37b-28ffacb55285"

func TestTenantIdDeterministic(t *testing.T) {
	id1 := TenantId("GMS", 83, 1)
	id2 := TenantId("GMS", 83, 1)
	if id1 != id2 {
		t.Fatalf("TenantId is not deterministic: %v != %v", id1, id2)
	}
}

func TestTenantIdUniqueness(t *testing.T) {
	gms83 := TenantId("GMS", 83, 1)
	gms84 := TenantId("GMS", 84, 1)
	jms83 := TenantId("JMS", 83, 1)

	if gms83 == gms84 {
		t.Fatalf("TenantId(GMS,83,1) == TenantId(GMS,84,1): %v", gms83)
	}
	if gms83 == jms83 {
		t.Fatalf("TenantId(GMS,83,1) == TenantId(JMS,83,1): %v", gms83)
	}
}

func TestTenantIdNotNilOrSentinel(t *testing.T) {
	id := TenantId("GMS", 83, 1)
	if id == uuid.Nil {
		t.Fatalf("TenantId returned uuid.Nil")
	}
	sentinel := uuid.MustParse(TenantUUID)
	if id == sentinel {
		t.Fatalf("TenantId returned the sentinel TenantUUID %v", sentinel)
	}
}

func TestIsCanonical(t *testing.T) {
	id := TenantId("GMS", 84, 1)

	if !IsCanonical(id, "GMS", 84, 1) {
		t.Fatalf("IsCanonical(TenantId(GMS,84,1), GMS, 84, 1) should be true")
	}
	if IsCanonical(id, "GMS", 83, 1) {
		t.Fatalf("IsCanonical(TenantId(GMS,84,1), GMS, 83, 1) should be false")
	}
	if IsCanonical(uuid.New(), "GMS", 84, 1) {
		t.Fatalf("IsCanonical(random uuid, GMS, 84, 1) should be false")
	}
}

func TestTenantIdDeterminismPin(t *testing.T) {
	// The literal below is the frozen canonical id for GMS 83.1.
	// If this test fails, Namespace or the format string changed — which
	// would orphan every canonical row in every environment.
	got := TenantId("GMS", 83, 1).String()
	if got != frozenGMS83dot1 {
		t.Fatalf("determinism pin broken: got %s, want %s", got, frozenGMS83dot1)
	}
}
