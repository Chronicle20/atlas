package slot

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:   "field1",
		name: "field2",
		wz:   "field3",
		slot: 44,
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
