package character_test

import (
	"atlas-login/character"
	"testing"
)

func TestBuilder_Build(t *testing.T) {
	m := character.NewBuilder().
		SetId(1000).
		SetAccountId(100).
		SetWorldId(1).
		SetName("TestChar").
		SetGender(1).
		SetSkinColor(0).
		SetFace(20000).
		SetHair(30000).
		SetLevel(50).
		SetJobId(111).
		SetStrength(10).
		SetDexterity(15).
		SetIntelligence(20).
		SetLuck(25).
		SetHp(1000).
		SetMaxHp(1500).
		SetMp(500).
		SetMaxMp(800).
		SetAp(5).
		SetSp("0,0,0,0").
		SetExperience(50000).
		SetFame(10).
		SetMapId(100000000).
		SetMeso(1000000).
		Build()

	if m.Id() != 1000 {
		t.Errorf("Id() = %d, want 1000", m.Id())
	}
	if m.AccountId() != 100 {
		t.Errorf("AccountId() = %d, want 100", m.AccountId())
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId() = %d, want 1", m.WorldId())
	}
	if m.Name() != "TestChar" {
		t.Errorf("Name() = %s, want 'TestChar'", m.Name())
	}
	if m.Gender() != 1 {
		t.Errorf("Gender() = %d, want 1", m.Gender())
	}
	if m.Level() != 50 {
		t.Errorf("Level() = %d, want 50", m.Level())
	}
	if m.JobId() != 111 {
		t.Errorf("JobId() = %d, want 111", m.JobId())
	}
	if m.Strength() != 10 {
		t.Errorf("Strength() = %d, want 10", m.Strength())
	}
	if m.Dexterity() != 15 {
		t.Errorf("Dexterity() = %d, want 15", m.Dexterity())
	}
	if m.Intelligence() != 20 {
		t.Errorf("Intelligence() = %d, want 20", m.Intelligence())
	}
	if m.Luck() != 25 {
		t.Errorf("Luck() = %d, want 25", m.Luck())
	}
	if m.Hp() != 1000 {
		t.Errorf("Hp() = %d, want 1000", m.Hp())
	}
	if m.MaxHp() != 1500 {
		t.Errorf("MaxHp() = %d, want 1500", m.MaxHp())
	}
	if m.Experience() != 50000 {
		t.Errorf("Experience() = %d, want 50000", m.Experience())
	}
	if m.Fame() != 10 {
		t.Errorf("Fame() = %d, want 10", m.Fame())
	}
	if m.MapId() != 100000000 {
		t.Errorf("MapId() = %d, want 100000000", m.MapId())
	}
}

func TestModel_ToBuilder(t *testing.T) {
	original := character.NewBuilder().
		SetId(1000).
		SetAccountId(100).
		SetWorldId(1).
		SetName("TestChar").
		SetLevel(50).
		SetJobId(111).
		SetHp(1000).
		SetMaxHp(1500).
		Build()

	// Clone and modify level
	cloned := original.ToBuilder().
		SetLevel(51).
		SetHp(1100).
		Build()

	// Original should be unchanged
	if original.Level() != 50 {
		t.Errorf("Original Level() = %d, want 50", original.Level())
	}
	if original.Hp() != 1000 {
		t.Errorf("Original Hp() = %d, want 1000", original.Hp())
	}

	// Cloned should have new values
	if cloned.Level() != 51 {
		t.Errorf("Cloned Level() = %d, want 51", cloned.Level())
	}
	if cloned.Hp() != 1100 {
		t.Errorf("Cloned Hp() = %d, want 1100", cloned.Hp())
	}

	// Other fields should be preserved
	if cloned.Id() != 1000 {
		t.Errorf("Cloned Id() = %d, want 1000", cloned.Id())
	}
	if cloned.Name() != "TestChar" {
		t.Errorf("Cloned Name() = %s, want 'TestChar'", cloned.Name())
	}
	if cloned.JobId() != 111 {
		t.Errorf("Cloned JobId() = %d, want 111", cloned.JobId())
	}
}

func TestNewBuilder_DefaultValues(t *testing.T) {
	m := character.NewBuilder().Build()

	if m.Id() != 0 {
		t.Errorf("Default Id() = %d, want 0", m.Id())
	}
	if m.Name() != "" {
		t.Errorf("Default Name() = %s, want ''", m.Name())
	}
	if m.Level() != 0 {
		t.Errorf("Default Level() = %d, want 0", m.Level())
	}
	if m.JobId() != 0 {
		t.Errorf("Default JobId() = %d, want 0", m.JobId())
	}
}

func TestCharacter_Gm(t *testing.T) {
	// GM character
	gm := character.NewBuilder().
		SetGm(1).
		Build()

	if !gm.Gm() {
		t.Error("Character with Gm=1 should be Gm()")
	}

	// Non-GM character
	normal := character.NewBuilder().
		SetGm(0).
		Build()

	if normal.Gm() {
		t.Error("Character with Gm=0 should not be Gm()")
	}
}

func TestCharacter_HasSPTable(t *testing.T) {
	testCases := []struct {
		jobId    uint16
		expected bool
	}{
		{2001, true},
		{2200, true},
		{2210, true},
		{2218, true},
		{0, false},
		{100, false},
		{1000, false},
	}

	for _, tc := range testCases {
		m := character.NewBuilder().SetJobId(tc.jobId).Build()
		if m.HasSPTable() != tc.expected {
			t.Errorf("JobId %d: HasSPTable() = %v, want %v", tc.jobId, m.HasSPTable(), tc.expected)
		}
	}
}
