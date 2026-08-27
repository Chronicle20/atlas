package pet

import (
	"reflect"
	"testing"
)

func TestExtractPopulatesName(t *testing.T) {
	rm := RestModel{Id: 7, TemplateId: 5000029, Name: "Fluffy", Level: 20, Slot: 0}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Name() != "Fluffy" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Fluffy")
	}
	if m.TemplateId() != 5000029 || m.Level() != 20 || !m.IsSpawned() {
		t.Errorf("other fields wrong: %+v", m)
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(7, 5000029, "Fluffy", 20, 3)

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
