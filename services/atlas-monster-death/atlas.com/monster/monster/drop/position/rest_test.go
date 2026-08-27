package position

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder().SetX(11).SetY(22).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
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
