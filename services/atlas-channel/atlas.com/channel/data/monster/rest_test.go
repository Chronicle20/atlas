package monster

import "testing"

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
