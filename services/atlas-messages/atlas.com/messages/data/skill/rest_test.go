package skill

import (
	"atlas-messages/data/skill/effect"
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	e, err := effect.Extract(effect.RestModel{WeaponAttack: 10, Hp: 5})
	if err != nil {
		t.Fatalf("effect.Extract failed: %v", err)
	}

	m := Model{
		id:            1,
		name:          "Power Strike",
		action:        true,
		element:       "fire",
		animationTime: 500,
		effects:       []effect.Model{e},
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
		t.Errorf("round trip mismatch.\nExpected %+v\nGot %+v", m, got)
	}
}
