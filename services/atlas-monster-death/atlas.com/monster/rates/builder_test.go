package rates

import (
	"testing"
)

func TestNewBuilder_DefaultsToUnitRates(t *testing.T) {
	m := NewBuilder().Build()

	if m.ExpRate() != 1.0 {
		t.Errorf("expected ExpRate() == 1.0, got %v", m.ExpRate())
	}
	if m.MesoRate() != 1.0 {
		t.Errorf("expected MesoRate() == 1.0, got %v", m.MesoRate())
	}
	if m.ItemDropRate() != 1.0 {
		t.Errorf("expected ItemDropRate() == 1.0, got %v", m.ItemDropRate())
	}
	if m.QuestExpRate() != 1.0 {
		t.Errorf("expected QuestExpRate() == 1.0, got %v", m.QuestExpRate())
	}
}

func TestBuilder_SetsEachRate(t *testing.T) {
	m := NewBuilder().SetExpRate(2.5).SetMesoRate(3.0).SetItemDropRate(4.0).SetQuestExpRate(5.0).Build()

	if m.ExpRate() != 2.5 {
		t.Errorf("expected ExpRate() == 2.5, got %v", m.ExpRate())
	}
	if m.MesoRate() != 3.0 {
		t.Errorf("expected MesoRate() == 3.0, got %v", m.MesoRate())
	}
	if m.ItemDropRate() != 4.0 {
		t.Errorf("expected ItemDropRate() == 4.0, got %v", m.ItemDropRate())
	}
	if m.QuestExpRate() != 5.0 {
		t.Errorf("expected QuestExpRate() == 5.0, got %v", m.QuestExpRate())
	}
}
