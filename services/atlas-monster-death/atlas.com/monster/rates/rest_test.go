package rates

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		expRate:      1.5,
		mesoRate:     2.5,
		itemDropRate: 3.5,
		questExpRate: 4.5,
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}
