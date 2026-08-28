package skill

import (
	"atlas-character/data/skill/effect"
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	er, err := effect.Extract(effect.RestModel{WeaponAttack: 5})
	if err != nil {
		t.Fatalf("effect.Extract failed: %v", err)
	}

	m := Model{
		id:            100,
		action:        true,
		element:       "fire",
		animationTime: 500,
		effects:       []effect.Model{er},
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
