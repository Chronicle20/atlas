package information

import "testing"

func TestExtract_CarriesLevelAndName(t *testing.T) {
	m, err := Extract(RestModel{Id: 100100, Name: "Blue Snail", Hp: 8, Experience: 3, Level: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Hp() != 8 {
		t.Errorf("expected Hp() == 8, got %d", m.Hp())
	}
	if m.Experience() != 3 {
		t.Errorf("expected Experience() == 3, got %d", m.Experience())
	}
	if m.Level() != 2 {
		t.Errorf("expected Level() == 2, got %d", m.Level())
	}
	if m.Name() != "Blue Snail" {
		t.Errorf("expected Name() == \"Blue Snail\", got %q", m.Name())
	}
}

func TestBuilder_SetsLevelAndName(t *testing.T) {
	m, err := NewBuilder().SetHp(1000).SetExperience(500).SetLevel(125).SetName("Zakum").Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Hp() != 1000 {
		t.Errorf("expected Hp() == 1000, got %d", m.Hp())
	}
	if m.Experience() != 500 {
		t.Errorf("expected Experience() == 500, got %d", m.Experience())
	}
	if m.Level() != 125 {
		t.Errorf("expected Level() == 125, got %d", m.Level())
	}
	if m.Name() != "Zakum" {
		t.Errorf("expected Name() == \"Zakum\", got %q", m.Name())
	}
}
