package monster

import "testing"

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
