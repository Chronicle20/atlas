package job

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestMpEaterSkillId(t *testing.T) {
	cases := []struct {
		name   string
		jobId  Id
		wantId skill.Id
		wantOk bool
	}{
		{"Magician (200) — no MP Eater", MagicianId, 0, false},
		{"FPWizard (210)", FirePoisonWizardId, skill.FirePoisonWizardMpEaterId, true},
		{"FPMage (211)", FirePoisonMagicianId, skill.FirePoisonWizardMpEaterId, true},
		{"FPArchMage (212)", FirePoisonArchMagicianId, skill.FirePoisonWizardMpEaterId, true},
		{"ILWizard (220)", IceLightningWizardId, skill.IceLightningWizardMpEaterId, true},
		{"ILMage (221)", IceLightningMagicianId, skill.IceLightningWizardMpEaterId, true},
		{"ILArchMage (222)", IceLightningArchMagicianId, skill.IceLightningWizardMpEaterId, true},
		{"Cleric (230)", ClericId, skill.ClericMpEaterId, true},
		{"Priest (231)", PriestId, skill.ClericMpEaterId, true},
		{"Bishop (232)", BishopId, skill.ClericMpEaterId, true},
		{"Fighter (110) — no MP Eater", FighterId, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotId, gotOk := MpEaterSkillId(tc.jobId)
			if gotOk != tc.wantOk {
				t.Fatalf("MpEaterSkillId(%v) ok = %v; want %v", tc.jobId, gotOk, tc.wantOk)
			}
			if tc.wantOk && gotId != tc.wantId {
				t.Fatalf("MpEaterSkillId(%v) id = %v; want %v", tc.jobId, gotId, tc.wantId)
			}
		})
	}
}
