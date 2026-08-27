package rate

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		characterId:  123,
		expRate:      1.5,
		mesoRate:     2.5,
		itemDropRate: 3.5,
		questExpRate: 4.5,
		factors: []Factor{
			{source: "event", rateType: "exp", multiplier: 2.0},
			{source: "item", rateType: "meso", multiplier: 1.2},
		},
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
