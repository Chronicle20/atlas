package pet

import "testing"

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
