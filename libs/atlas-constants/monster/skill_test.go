package monster

import (
	"sort"
	"testing"
)

func TestReflectKindConstants(t *testing.T) {
	if ReflectKindPhysical != "PHYSICAL" {
		t.Fatalf("ReflectKindPhysical = %q, want PHYSICAL", ReflectKindPhysical)
	}
	if ReflectKindMagical != "MAGICAL" {
		t.Fatalf("ReflectKindMagical = %q, want MAGICAL", ReflectKindMagical)
	}
}

func TestReflectKindForSkill(t *testing.T) {
	cases := []struct {
		skillId uint16
		want    string
	}{
		{SkillTypePhysicalCounter, ReflectKindPhysical},
		{SkillTypeMagicCounter, ReflectKindMagical},
		{SkillTypePhysicalMagicCounter, ReflectKindPhysical}, // physical+magic combined; physical wins for the gate
		{1, ""},
	}
	for _, c := range cases {
		got := ReflectKindForSkill(c.skillId)
		if got != c.want {
			t.Fatalf("ReflectKindForSkill(%d) = %q, want %q", c.skillId, got, c.want)
		}
	}
}

func TestSkillTypeToStatusName_Carnival(t *testing.T) {
	cases := []struct {
		skillType uint16
		want      TemporaryStatType
	}{
		{SkillTypeCarnivalPAD, TemporaryStatTypePowerUp},
		{SkillTypeCarnivalMAD, TemporaryStatTypeMagicUp},
		{SkillTypeCarnivalPDR, TemporaryStatTypePowerGuardUp},
		{SkillTypeCarnivalMDR, TemporaryStatTypeMagicGuardUp},
		{SkillTypeCarnivalACC, TemporaryStatTypeAccuracy},
		{SkillTypeCarnivalEVA, TemporaryStatTypeAvoidability},
		{SkillTypeCarnivalSpeed, TemporaryStatTypeSpeed},
		{SkillTypeCarnivalSealSkill, TemporaryStatTypeSealSkill},
		{149, ""},
		{158, ""},
		{SkillTypePhysicalMagicCounter, ""},
	}
	for _, c := range cases {
		got := SkillTypeToStatusName(c.skillType)
		if got != c.want {
			t.Fatalf("SkillTypeToStatusName(%d) = %q, want %q", c.skillType, got, c.want)
		}
	}
}

func TestSkillTypeToStatusName_SharedStatArms(t *testing.T) {
	cases := []struct {
		skillType uint16
		want      TemporaryStatType
	}{
		{100, TemporaryStatTypePowerUp},
		{110, TemporaryStatTypePowerUp},
		{150, TemporaryStatTypePowerUp},
		{101, TemporaryStatTypeMagicUp},
		{111, TemporaryStatTypeMagicUp},
		{151, TemporaryStatTypeMagicUp},
		{102, TemporaryStatTypePowerGuardUp},
		{112, TemporaryStatTypePowerGuardUp},
		{152, TemporaryStatTypePowerGuardUp},
		{103, TemporaryStatTypeMagicGuardUp},
		{113, TemporaryStatTypeMagicGuardUp},
		{153, TemporaryStatTypeMagicGuardUp},
		{115, TemporaryStatTypeSpeed},
		{156, TemporaryStatTypeSpeed},
	}
	for _, c := range cases {
		got := SkillTypeToStatusName(c.skillType)
		if got != c.want {
			t.Fatalf("SkillTypeToStatusName(%d) = %q, want %q", c.skillType, got, c.want)
		}
	}
}

func TestIsAoeSkill_CarnivalAndRegressions(t *testing.T) {
	cases := []struct {
		skillType uint16
		want      bool
	}{
		{150, true},
		{151, true},
		{152, true},
		{153, true},
		{154, true},
		{155, true},
		{156, true},
		{157, true},
		{SkillTypeWeaponAttackUpAoe, true},
		{SkillTypeMagicAttackUpAoe, true},
		{SkillTypeWeaponDefenseUpAoe, true},
		{SkillTypeMagicDefenseUpAoe, true},
		{SkillTypeHeal, true},
		{100, false},
		{101, false},
		{102, false},
		{103, false},
		{115, false},
		{120, false},
		{140, false},
		{143, false},
		{145, false},
		{200, false},
		{149, false},
		{158, false},
	}
	for _, c := range cases {
		got := IsAoeSkill(c.skillType)
		if got != c.want {
			t.Fatalf("IsAoeSkill(%d) = %v, want %v", c.skillType, got, c.want)
		}
	}
}

func TestSkillNameToId_Carnival(t *testing.T) {
	cases := []struct {
		name   string
		wantId uint16
		wantOk bool
	}{
		{"CARNIVAL_PAD", 150, true},
		{"CARNIVAL_MAD", 151, true},
		{"CARNIVAL_PDR", 152, true},
		{"CARNIVAL_MDR", 153, true},
		{"CARNIVAL_ACC", 154, true},
		{"CARNIVAL_EVA", 155, true},
		{"CARNIVAL_SPEED", 156, true},
		{"CARNIVAL_SEAL_SKILL", 157, true},
		{"CARNIVAL_NOPE", 0, false},
	}
	for _, c := range cases {
		gotId, gotOk := SkillNameToId(c.name)
		if gotId != c.wantId || gotOk != c.wantOk {
			t.Fatalf("SkillNameToId(%q) = (%d, %v), want (%d, %v)", c.name, gotId, gotOk, c.wantId, c.wantOk)
		}
	}
}

func TestSkillTypeNames_IncludesCarnival(t *testing.T) {
	want := []string{
		"CARNIVAL_PAD",
		"CARNIVAL_MAD",
		"CARNIVAL_PDR",
		"CARNIVAL_MDR",
		"CARNIVAL_ACC",
		"CARNIVAL_EVA",
		"CARNIVAL_SPEED",
		"CARNIVAL_SEAL_SKILL",
	}
	names := SkillTypeNames()
	if !sort.StringsAreSorted(names) {
		t.Fatalf("SkillTypeNames() = %v, want sorted ascending", names)
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Fatalf("SkillTypeNames() missing %q", w)
		}
	}
}

func TestSkillCategory_Carnival(t *testing.T) {
	cases := []struct {
		skillType uint16
		want      string
	}{
		{SkillTypeCarnivalPAD, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalMAD, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalPDR, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalMDR, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalACC, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalEVA, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalSpeed, SkillCategoryCarnivalBuf},
		{SkillTypeCarnivalSealSkill, SkillCategoryCarnivalBuf},
		{149, ""},
		{158, ""},
	}
	for _, c := range cases {
		got := SkillCategory(c.skillType)
		if got != c.want {
			t.Fatalf("SkillCategory(%d) = %q, want %q", c.skillType, got, c.want)
		}
	}
}
