package reactor

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:             11,
		classification: 22,
		name:           "field3",
		x:              44,
		y:              55,
		delay:          66,
		direction:      77,
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
