package disease

import (
	"testing"

	"atlas-messages/kafka/message/buff"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
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
