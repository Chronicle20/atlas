package monster

import "testing"

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
