package character

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:   11,
		name: "field2",
		gm:   33,
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
