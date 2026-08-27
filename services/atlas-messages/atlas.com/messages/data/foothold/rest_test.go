package foothold

import (
	"reflect"
	"testing"
)

func TestRestModel_GetName(t *testing.T) {
	if (RestModel{}).GetName() != "footholds" {
		t.Errorf("GetName() = %q, want %q", (RestModel{}).GetName(), "footholds")
	}
}

func TestPositionRestModel_GetName(t *testing.T) {
	if (PositionRestModel{}).GetName() != "positions" {
		t.Errorf("GetName() = %q, want %q", (PositionRestModel{}).GetName(), "positions")
	}
}

func TestExtract_Id(t *testing.T) {
	m, err := Extract(RestModel{Id: 42})
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if m.Id() != 42 {
		t.Errorf("Id() = %d, want 42", m.Id())
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id: 11,
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
