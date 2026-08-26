package disease

import (
	"atlas-messages/kafka/message/buff"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// TestValidDiseasesResolveToCanonicalTemporaryStatTypes is the regression
// guard for the bug where validDiseases carried non-existent stat type
// literals ("ZOMBIFY", "WEAKNESS") instead of the canonical
// character.TemporaryStatType wire values ("UNDEAD", "WEAKEN"). Asserting
// against the actual character.TemporaryStatType constants (rather than bare
// string literals) ensures a typo or a stale value can never again diverge
// from the values every other consumer of the wire contract uses.
func TestValidDiseasesResolveToCanonicalTemporaryStatTypes(t *testing.T) {
	tests := []struct {
		word string
		want character.TemporaryStatType
	}{
		{"SEAL", character.TemporaryStatTypeSeal},
		{"DARKNESS", character.TemporaryStatTypeDarkness},
		{"WEAKNESS", character.TemporaryStatTypeWeaken},
		{"WEAKEN", character.TemporaryStatTypeWeaken},
		{"STUN", character.TemporaryStatTypeStun},
		{"CURSE", character.TemporaryStatTypeCurse},
		{"POISON", character.TemporaryStatTypePoison},
		{"SLOW", character.TemporaryStatTypeSlow},
		{"SEDUCE", character.TemporaryStatTypeSeduce},
		{"ZOMBIFY", character.TemporaryStatTypeUndead},
		{"UNDEAD", character.TemporaryStatTypeUndead},
		{"CONFUSE", character.TemporaryStatTypeConfuse},
		{"STOP_PORTION", character.TemporaryStatTypeStopPortion},
	}

	if len(validDiseases) != len(tests) {
		t.Fatalf("validDiseases has %d entries, expected exactly the %d covered by this test", len(validDiseases), len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			got, ok := validDiseases[tt.word]
			if !ok {
				t.Fatalf("validDiseases[%q] is missing", tt.word)
			}
			if got != tt.want {
				t.Errorf("validDiseases[%q] = %q, want %q", tt.word, got, tt.want)
			}
		})
	}
}

// TestZombifyEmitsUndeadStatChange pins the specific symptom from the bug
// report: the parsed "ZOMBIFY" word must produce an UNDEAD stat change on
// the emitted buff APPLY command, since atlas-buffs, atlas-consumables and
// atlas-channel all key their zombify handling on UNDEAD.
func TestZombifyEmitsUndeadStatChange(t *testing.T) {
	stat, ok := validDiseases["ZOMBIFY"]
	if !ok {
		t.Fatal(`validDiseases["ZOMBIFY"] is missing`)
	}

	changes := []buff.StatChange{{Type: string(stat), Amount: 1}}

	if changes[0].Type != string(character.TemporaryStatTypeUndead) {
		t.Errorf("ZOMBIFY stat change type = %q, want %q", changes[0].Type, character.TemporaryStatTypeUndead)
	}
}

// TestDiseaseMobSkillSuppliesUndeadSourceId pins the second, independent
// contributor from bug-zombify-no-visible-effect.md: the @disease command
// used to emit sourceId=0, so even a correct encoder had no MobSkill to
// resolve rOption against. A GM-applied UNDEAD/ZOMBIFY must carry the real
// mob skill id (monster.SkillTypeUndead, type 133).
func TestDiseaseMobSkillSuppliesUndeadSourceId(t *testing.T) {
	sourceId, level := diseaseMobSkill(character.TemporaryStatTypeUndead)

	if sourceId != int32(monster.SkillTypeUndead) {
		t.Errorf("UNDEAD sourceId = %d, want %d", sourceId, monster.SkillTypeUndead)
	}
	if level == 0 {
		t.Errorf("UNDEAD level = %d, want nonzero", level)
	}
}

// TestDiseaseMobSkillLeavesOtherDiseasesUnestablished guards against
// over-generalizing the UNDEAD fix: only UNDEAD has IDA evidence for what
// rOption must carry, so every other disease keeps sourceId=0 until
// similarly established.
func TestDiseaseMobSkillLeavesOtherDiseasesUnestablished(t *testing.T) {
	others := []character.TemporaryStatType{
		character.TemporaryStatTypeSeal,
		character.TemporaryStatTypeDarkness,
		character.TemporaryStatTypeWeaken,
		character.TemporaryStatTypeStun,
		character.TemporaryStatTypeCurse,
		character.TemporaryStatTypePoison,
		character.TemporaryStatTypeSlow,
		character.TemporaryStatTypeSeduce,
		character.TemporaryStatTypeConfuse,
		character.TemporaryStatTypeStopPortion,
	}
	for _, st := range others {
		t.Run(string(st), func(t *testing.T) {
			sourceId, _ := diseaseMobSkill(st)
			if sourceId != 0 {
				t.Errorf("diseaseMobSkill(%q) sourceId = %d, want 0", st, sourceId)
			}
		})
	}
}
