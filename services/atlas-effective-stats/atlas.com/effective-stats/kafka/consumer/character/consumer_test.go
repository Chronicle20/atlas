package character

import (
	"atlas-effective-stats/stat"
	"testing"

	character2 "atlas-effective-stats/character"
	"atlas-effective-stats/external/data/equipment"
	character3 "atlas-effective-stats/kafka/message/character"
	conststat "github.com/Chronicle20/atlas/libs/atlas-constants/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
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

func TestHandleStatChanged_JobUpdateRefetchesWearer(t *testing.T) {
	t.Skip("integration test placeholder; covered by Task 21 — keeps the consumer split honest at unit scope")
}

func TestHandleStatChanged_LevelUpdateRefetchesWearer(t *testing.T) {
	t.Skip("integration test placeholder; covered by Task 21")
}

func TestHandleStatChanged_NumericAndProfileBothInOneEvent(t *testing.T) {
	t.Skip("integration test placeholder; covered by Task 21")
}

// TestHandleStatChanged_LuckRise_RegatesEquipment is the consumer-level
// reproduction of PRD §4.1 follow-up: a Pole Arm gated on LUK 40 should
// reactivate as soon as the wearer's LUK ticks from 39 to 40 via a
// STAT_CHANGED event, with MaxMp jumping by the asset's +50 bonus.
func TestHandleStatChanged_LuckRise_RegatesEquipment(t *testing.T) {
	setupCharacterTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	stubs := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 39,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1052095: {reqLuk: 40}},
	})
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	if _, _, err := character2.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345); err != nil {
		t.Fatalf("GetEffectiveStats: %v", err)
	}
	m, _ := character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6330 {
		t.Fatalf("pre: MaxMp = %d, want 6330", m.Computed().MaxMp())
	}

	handleStatChanged(l, ctx, character3.StatusEvent[character3.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character3.StatusEventTypeStatChanged,
		Body: character3.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []conststat.Type{conststat.TypeLuck},
			Values:    map[string]interface{}{"luck": 40},
		},
	})

	m, _ = character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("post: MaxMp = %d, want 6380 (asset reactivated)", m.Computed().MaxMp())
	}
}

// TestHandleStatChanged_JobChange_RefetchesAndRegates validates that a
// TypeJob STAT_CHANGED event with Values=nil triggers a wearer refetch and
// re-runs equipment qualification — an asset gated on reqJob=1 (Warrior
// branch) must activate once the wearer transitions from Magician (200) to
// Warrior (100).
func TestHandleStatChanged_JobChange_RefetchesAndRegates(t *testing.T) {
	setupCharacterTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	cfg := stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200, str: 4, dex: 25, intl: 4, luk: 50,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 99, templateId: 1402000, slot: -10, str: 5,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1402000: {reqJob: 1}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	_, _, _ = character2.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345)
	m, _ := character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != uint32(cfg.character.str) {
		t.Fatalf("pre: STR = %d, want %d", m.Computed().Strength(), cfg.character.str)
	}

	cfg.character.jobId = 100
	stubs.character.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleStatChanged(l, ctx, character3.StatusEvent[character3.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character3.StatusEventTypeStatChanged,
		Body: character3.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []conststat.Type{conststat.TypeJob},
			Values:    nil,
		},
	})

	m, _ = character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != uint32(cfg.character.str+5) {
		t.Errorf("post-job-change: STR = %d, want %d", m.Computed().Strength(), cfg.character.str+5)
	}
}

// TestHandleStatChanged_LevelRise_RefetchesAndRegates validates that a
// TypeLevel STAT_CHANGED event with Values=nil triggers a wearer refetch and
// re-runs equipment qualification — an asset gated on reqLevel=30 must
// activate once the wearer dings from level 29 to 30.
func TestHandleStatChanged_LevelRise_RefetchesAndRegates(t *testing.T) {
	setupCharacterTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	cfg := stubConfig{
		character: stubCharacter{
			level: 29, jobId: 100, str: 50, dex: 4, intl: 4, luk: 4,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 7, templateId: 1302000, slot: -10, str: 5,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1302000: {reqLevel: 30}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	_, _, _ = character2.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345)
	m, _ := character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != 50 {
		t.Fatalf("pre: STR = %d, want 50", m.Computed().Strength())
	}

	cfg.character.level = 30
	stubs.character.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleStatChanged(l, ctx, character3.StatusEvent[character3.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character3.StatusEventTypeStatChanged,
		Body: character3.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []conststat.Type{conststat.TypeLevel},
			Values:    nil,
		},
	})

	m, _ = character2.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != 55 {
		t.Errorf("post-level: STR = %d, want 55", m.Computed().Strength())
	}
}
