package character

import (
	"atlas-effective-stats/stat"
	"testing"
)

func TestEquippedAsset_Getters(t *testing.T) {
	bonuses := []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	}
	a := NewEquippedAsset(42, 1052095, bonuses)

	if a.AssetId() != 42 {
		t.Errorf("AssetId() = %d, want 42", a.AssetId())
	}
	if a.TemplateId() != 1052095 {
		t.Errorf("TemplateId() = %d, want 1052095", a.TemplateId())
	}
	got := a.Bonuses()
	if len(got) != 1 || got[0].Amount() != 50 {
		t.Errorf("Bonuses() = %+v, want one MaxMp=50", got)
	}
}

func TestEquippedAsset_BonusesIsDefensiveCopy(t *testing.T) {
	bonuses := []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	}
	a := NewEquippedAsset(42, 1052095, bonuses)

	bonuses[0] = stat.NewBonus("equipment:42", stat.TypeMaxMp, 9999)
	if a.Bonuses()[0].Amount() != 50 {
		t.Errorf("internal bonuses leaked through constructor; got %d", a.Bonuses()[0].Amount())
	}

	out := a.Bonuses()
	out[0] = stat.NewBonus("equipment:42", stat.TypeMaxMp, -1)
	if a.Bonuses()[0].Amount() != 50 {
		t.Errorf("internal bonuses leaked through Bonuses(); got %d", a.Bonuses()[0].Amount())
	}
}
