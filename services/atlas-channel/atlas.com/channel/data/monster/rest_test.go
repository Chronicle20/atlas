package monster

import (
	"reflect"
	"testing"
)

func TestExtract(t *testing.T) {
	rm := RestModel{Boss: true, FixedDamage: 5}
	if err := rm.SetID("8510000"); err != nil {
		t.Fatal(err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if m.Id() != 8510000 {
		t.Errorf("Id=%d, want 8510000", m.Id())
	}
	if !m.Boss() {
		t.Error("Boss=false, want true")
	}
	if m.FixedDamage() != 5 {
		t.Errorf("FixedDamage=%d, want 5", m.FixedDamage())
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:          11,
		boss:        true,
		fixedDamage: 33,
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
