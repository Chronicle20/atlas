package skill

import (
	"atlas-channel/data/skill/effect"
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel, including the nested
// effect slice, survives a Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	em, err := effect.Extract(effect.RestModel{
		WeaponAttack: 10,
		MagicAttack:  20,
		Duration:     3000,
	})
	if err != nil {
		t.Fatalf("effect.Extract failed: %v", err)
	}

	m := Model{
		id:            1004,
		action:        true,
		element:       "F",
		animationTime: 500,
		effects:       []effect.Model{em},
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
