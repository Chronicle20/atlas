package monster

import (
	"reflect"
	"testing"
)

func TestRestModel_GetName(t *testing.T) {
	if (RestModel{}).GetName() != "monsters" {
		t.Errorf("GetName() = %q, want %q", (RestModel{}).GetName(), "monsters")
	}
}

func TestExtract_IdAndName(t *testing.T) {
	rm := RestModel{Id: 100100, Name: "Snail"}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if m.Id() != 100100 {
		t.Errorf("Id() = %d, want 100100", m.Id())
	}
	if m.Name() != "Snail" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Snail")
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:   11,
		name: "field2",
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
