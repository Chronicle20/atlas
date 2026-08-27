package effect

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip verifies Transform is the faithful inverse of
// Extract: Extract(Transform(m)) reproduces m.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		WeaponAttack:  10,
		MagicAttack:   20,
		Hp:            30,
		Duration:      40000,
		X:             50,
		Y:             60,
		Prop:          0.75,
		MonsterStatus: map[string]uint32{"stun": 1},
		Statups: []StatupRestModel{
			{Type: "WEAPON_DEFENSE", Amount: 60},
			{Type: "MAGIC_DEFENSE", Amount: 30},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (round trip) failed: %v", err)
	}

	if !reflect.DeepEqual(m2, m) {
		t.Errorf("round trip mismatch. want %+v, got %+v", m, m2)
	}
}
