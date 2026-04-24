package character

import (
	"atlas-effective-stats/stat"
	"testing"
)

// Regression: atlas-character emits STAT_CHANGED with only the keys that
// changed (e.g. {luck: 38} for a single AP into luck). The handler was
// treating Values as a full snapshot and building stat.NewBase with 0 for
// every absent field, which wiped MaxHP/MaxMP in the registry and later
// killed the character via the HP-regen clamp path in atlas-character.
func TestMergeBaseStats_PreservesFieldsMissingFromEvent(t *testing.T) {
	current := stat.NewBase(4, 4, 4, 4, 337, 165)

	// A single AP distribution into luck — only `luck` is present.
	values := map[string]interface{}{
		"luck": float64(38),
	}

	got := mergeBaseStats(current, values)

	if got.Strength() != 4 || got.Dexterity() != 4 || got.Intelligence() != 4 {
		t.Errorf("primary stats zeroed on partial event: STR=%d DEX=%d INT=%d, want all 4",
			got.Strength(), got.Dexterity(), got.Intelligence())
	}
	if got.Luck() != 38 {
		t.Errorf("Luck = %d, want 38", got.Luck())
	}
	if got.MaxHp() != 337 {
		t.Errorf("MaxHp zeroed on partial event: %d, want 337 (this is the regression that killed character 12)", got.MaxHp())
	}
	if got.MaxMp() != 165 {
		t.Errorf("MaxMp zeroed on partial event: %d, want 165", got.MaxMp())
	}
}

func TestMergeBaseStats_OverridesFieldsPresentInEvent(t *testing.T) {
	current := stat.NewBase(4, 4, 4, 4, 337, 165)

	// Level-up event carries the new MaxHp / MaxMp / INT.
	values := map[string]interface{}{
		"intelligence": float64(5),
		"max_hp":       float64(350),
		"max_mp":       float64(175),
	}

	got := mergeBaseStats(current, values)

	if got.Intelligence() != 5 {
		t.Errorf("Intelligence = %d, want 5", got.Intelligence())
	}
	if got.MaxHp() != 350 {
		t.Errorf("MaxHp = %d, want 350", got.MaxHp())
	}
	if got.MaxMp() != 175 {
		t.Errorf("MaxMp = %d, want 175", got.MaxMp())
	}
	// Unchanged primary stats preserved.
	if got.Strength() != 4 || got.Dexterity() != 4 || got.Luck() != 4 {
		t.Errorf("unchanged primary stats altered: STR=%d DEX=%d LUK=%d",
			got.Strength(), got.Dexterity(), got.Luck())
	}
}

func TestMergeBaseStats_EmptyValuesPreservesCurrent(t *testing.T) {
	current := stat.NewBase(4, 4, 38, 4, 337, 165)

	got := mergeBaseStats(current, map[string]interface{}{})

	if got != current {
		t.Errorf("empty values mutated base: got %+v, want %+v", got, current)
	}
}

// Simulates the exact sequence from the character-12 death. Starting with
// level-up MaxHp=337, every subsequent luck-up must leave MaxHp untouched.
func TestMergeBaseStats_LuckUpSequencePreservesMaxHp(t *testing.T) {
	base := stat.NewBase(4, 4, 4, 4, 337, 165)

	for luck := uint16(38); luck <= 42; luck++ {
		base = mergeBaseStats(base, map[string]interface{}{
			"luck": float64(luck),
		})
		if base.MaxHp() != 337 {
			t.Fatalf("MaxHp regressed to %d at luck=%d; regression would kill the character on next HoT tick", base.MaxHp(), luck)
		}
		if base.MaxMp() != 165 {
			t.Fatalf("MaxMp regressed to %d at luck=%d", base.MaxMp(), luck)
		}
		if base.Luck() != luck {
			t.Fatalf("Luck not advanced: got %d, want %d", base.Luck(), luck)
		}
	}
}

func TestToUint16_CoercesJSONNumericTypes(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want uint16
	}{
		{"float64 (JSON number)", float64(42), 42},
		{"int", int(42), 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"uint16", uint16(42), 42},
		{"uint32", uint32(42), 42},
		{"uint64", uint64(42), 42},
		{"unknown type returns zero", "not a number", 0},
		{"nil returns zero", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toUint16(tt.in); got != tt.want {
				t.Errorf("toUint16(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestMergeUint16_AbsentKeyReturnsCurrent(t *testing.T) {
	got := mergeUint16(337, map[string]interface{}{}, "max_hp")
	if got != 337 {
		t.Errorf("absent key returned %d; must preserve current=337 — this is the whole bug", got)
	}
}
