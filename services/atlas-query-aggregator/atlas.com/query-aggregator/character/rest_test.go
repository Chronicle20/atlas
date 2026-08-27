package character

import (
	"testing"
)

// TestExtract_SpawnPoint asserts the inbound seam: the RestModel value reaches
// the model. Anchored to the RestModel literal rather than to a second derived
// model -- an Extract/Transform idempotence assertion is blind to a dropped
// field, because a dropped field is zero on both sides of the comparison.
func TestExtract_SpawnPoint(t *testing.T) {
	rm := RestModel{Id: 1, Sp: "0", SpawnPoint: 11}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
}

// TestTransform_SpawnPointPreservesUint32 pins the outbound seam at full
// uint32 fidelity. This service re-serves spawnPoint over JSON:API rather than
// over the wire, so the value must NOT be narrowed to a byte here; 300 is
// above the byte ceiling precisely so a reintroduced cast fails loudly.
func TestTransform_SpawnPointPreservesUint32(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetSp("0").
		SetSpawnPoint(300).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.SpawnPoint != 300 {
		t.Errorf("SpawnPoint = %d, want 300", rm.SpawnPoint)
	}
}
