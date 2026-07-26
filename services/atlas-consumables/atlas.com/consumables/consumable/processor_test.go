package consumable

import (
	"atlas-consumables/asset"
	"atlas-consumables/character"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/equipable"
	"io"
	"testing"

	consumable3 "atlas-consumables/data/consumable"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestIsNotSlotConsumingScroll_SpikeScroll(t *testing.T) {
	// Spike scrolls are in the 2040200-2040299 range (shoe spikes)
	spikeScrollIds := []item.Id{2040727} // Cape for Cold Protection
	for _, id := range spikeScrollIds {
		if !item.IsScrollSpikes(id) && !item.IsScrollColdProtection(id) {
			// Skip IDs that aren't actually spikes in the item library
			continue
		}
		if item.IsScrollSpikes(id) {
			result := IsNotSlotConsumingScroll(id)
			if !result {
				t.Errorf("expected spike scroll %d to be non-slot consuming", id)
			}
		}
	}
}

func TestIsNotSlotConsumingScroll_ColdProtectionScroll(t *testing.T) {
	// Cold protection scrolls
	coldScrollIds := []item.Id{2040727} // Cape for Cold Protection
	for _, id := range coldScrollIds {
		if item.IsScrollColdProtection(id) {
			result := IsNotSlotConsumingScroll(id)
			if !result {
				t.Errorf("expected cold protection scroll %d to be non-slot consuming", id)
			}
		}
	}
}

func TestIsNotSlotConsumingScroll_RegularScroll(t *testing.T) {
	// Regular scroll that should consume slots
	regularScrollId := item.Id(2040001) // Regular scroll
	result := IsNotSlotConsumingScroll(regularScrollId)
	if result {
		t.Errorf("expected regular scroll %d to consume slots", regularScrollId)
	}
}

func TestRollStatAdjustment_ReturnsValidRange(t *testing.T) {
	// Roll multiple times and verify all results are in valid range
	counts := make(map[int16]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		result := rollStatAdjustment()
		if result < -5 || result > 5 {
			t.Errorf("rollStatAdjustment returned %d, expected range [-5, 5]", result)
		}
		counts[result]++
	}

	// Verify we got some distribution (not all same value)
	if len(counts) < 5 {
		t.Errorf("expected at least 5 different values, got %d", len(counts))
	}
}

func TestRollStatAdjustment_ZeroIsMostCommon(t *testing.T) {
	// Based on the probability distribution, 0 should be most common (~18.38%)
	counts := make(map[int16]int)
	iterations := 100000

	for i := 0; i < iterations; i++ {
		result := rollStatAdjustment()
		counts[result]++
	}

	// 0 should be the most common or close to it
	zeroCount := counts[0]
	zeroPercent := float64(zeroCount) / float64(iterations) * 100

	// Should be around 18.38%, allow some variance
	if zeroPercent < 15 || zeroPercent > 22 {
		t.Errorf("zero percent was %.2f%%, expected around 18.38%%", zeroPercent)
	}
}

func TestGenerateChaosChanges_SkipsZeroStats(t *testing.T) {
	// All zero stats - should produce no changes
	stats := []chaosStat{
		{0, equipable.AddStrength, 1},
		{0, equipable.AddDexterity, 1},
		{0, equipable.AddIntelligence, 1},
		{0, equipable.AddLuck, 1},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for all zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_GeneratesForNonZeroStats(t *testing.T) {
	// Non-zero stats should produce changes
	stats := []chaosStat{
		{10, equipable.AddStrength, 1},
		{0, equipable.AddDexterity, 1},
		{15, equipable.AddIntelligence, 1},
		{0, equipable.AddLuck, 1},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("expected 2 changes for 2 non-zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_AppliesMultiplier(t *testing.T) {
	stats := []chaosStat{
		{10, equipable.AddHp, 10},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 1 {
		t.Errorf("expected 1 change for HP stat, got %d", len(changes))
	}
}

func TestApplyChaos_AllStats(t *testing.T) {
	// Create asset with all stats non-zero using flat builder
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetWeaponAttack(10).
		SetWeaponDefense(10).
		SetMagicAttack(10).
		SetMagicDefense(10).
		SetAccuracy(10).
		SetAvoidability(10).
		SetSpeed(10).
		SetJump(10).
		SetHp(100).
		SetMp(100).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 14 changes (one for each non-zero stat)
	if len(changes) != 14 {
		t.Errorf("expected 14 changes, got %d", len(changes))
	}
}

func TestApplyChaos_PartialStats(t *testing.T) {
	// Create asset with only some stats non-zero
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetStrength(10).
		SetLuck(10).
		SetWeaponAttack(5).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 changes (str, luk, weapon attack)
	if len(changes) != 3 {
		t.Errorf("expected 3 changes for 3 non-zero stats, got %d", len(changes))
	}
}

func TestApplyChaos_HPMPMultiplier(t *testing.T) {
	// Test that HP/MP adjustments are multiplied by 10
	// Create asset with only HP and MP non-zero
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetHp(100).
		SetMp(100).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 changes (HP and MP only)
	if len(changes) != 2 {
		t.Errorf("expected 2 changes for HP/MP, got %d", len(changes))
	}

	// Apply changes to a builder and verify the multiplier effect
	builder := asset.NewBuilder(uuid.New(), 1000000).
		SetHp(100).
		SetMp(100)

	for _, change := range changes {
		change(builder)
	}

	result := builder.Build()

	// The change should be +/- 10, 20, 30, 40, or 50 (base adjustment * 10)
	hpDiff := int(result.Hp()) - 100
	mpDiff := int(result.Mp()) - 100

	// HP/MP changes should be multiples of 10
	if hpDiff%10 != 0 {
		t.Errorf("HP diff %d should be multiple of 10", hpDiff)
	}
	if mpDiff%10 != 0 {
		t.Errorf("MP diff %d should be multiple of 10", mpDiff)
	}

	// Changes should be in range [-50, 50]
	if hpDiff < -50 || hpDiff > 50 {
		t.Errorf("HP diff %d should be in range [-50, 50]", hpDiff)
	}
	if mpDiff < -50 || mpDiff > 50 {
		t.Errorf("MP diff %d should be in range [-50, 50]", mpDiff)
	}
}

// Test helper to create test equipable asset
func createTestEquipableAsset(templateId uint32, slots uint16, level byte) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetId(1).
		SetSlots(slots).
		SetLevel(level).
		Build()
}

// Test helper to create test scroll asset
func createTestScrollAsset(templateId uint32) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetId(2).
		Build()
}

func makeCureModel(t *testing.T, specs map[consumable3.SpecType]int32) consumable3.Model {
	t.Helper()
	rm := consumable3.RestModel{Spec: specs}
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

func TestCollectCureTypes_AntidotePot(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON"}, got)
}

func TestCollectCureTypes_HolyWater(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeSeal:  1,
		consumable3.SpecTypeCurse: 1,
	})
	got := collectCureTypes(ci)
	// Order is fixed (POISON, DARKNESS, WEAKEN, SEAL, CURSE) for determinism;
	// missing entries are dropped, so Holy Water yields just SEAL then CURSE.
	assert.Equal(t, []string{"SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_AllCure(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison:   1,
		consumable3.SpecTypeDarkness: 1,
		consumable3.SpecTypeWeakness: 1,
		consumable3.SpecTypeSeal:     1,
		consumable3.SpecTypeCurse:    1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON", "DARKNESS", "WEAKEN", "SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_NonCureConsumable(t *testing.T) {
	// White potion: HP recovery only, no cure flags.
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeHP: 1000,
	})
	got := collectCureTypes(ci)
	assert.Empty(t, got)
}

func TestCollectCureTypes_ZeroFlagsIgnored(t *testing.T) {
	// A 0-valued cure spec must be treated as "not present" (parser default).
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 0,
		consumable3.SpecTypeCurse:  1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"CURSE"}, got)
}

func TestUsesStandardConsumer(t *testing.T) {
	// Standard-consumer routing for items that need ApplyItemEffects (HP/MP
	// recovery, status buffs, status cure). Anything not matching here falls
	// through to ConsumeBare and silently skips effect application.
	cases := []struct {
		name   string
		itemId item.Id
		want   bool
	}{
		{"red potion (200)", item.Id(2000001), true},
		{"white potion (200)", item.Id(2000020), true},
		{"food/apple (201)", item.Id(2010000), true},
		{"hp food (202)", item.Id(2020000), true},
		{"return scroll (203)", item.Id(2030000), false},
		{"equip scroll (204)", item.Id(2040727), false},
		{"antidote — cure pot (205)", item.Id(2050001), true},
		{"all cure potion (205)", item.Id(2050004), true},
		{"morph potion (221)", item.Id(2210000), true},
		{"cliff's special potion — morphRandom (221)", item.Id(2211000), true},
		{"maplemas party potion (221; client intercepts 2212xxx before use-item, but classification routing is uniform)", item.Id(2212000), true},
		{"arrow (206)", item.Id(2060000), false},
		{"throwing star (207)", item.Id(2070000), false},
		{"summoning sack (210)", item.Id(2100000), false},
		{"pet food (212)", item.Id(2120000), false},
		{"weapon (130)", item.Id(1302000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := usesStandardConsumer(tc.itemId)
			assert.Equal(t, tc.want, got, "itemId %d (classification %d)", tc.itemId, item.GetClassification(tc.itemId))
		})
	}
}

func makeScrollModel(t *testing.T, success uint32, cursed uint32, incSTR uint32) consumable3.Model {
	t.Helper()
	rm := consumable3.RestModel{Success: success, Cursed: cursed, IncreaseSTR: incSTR}
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

func TestBuildScrollChanges_RegularSuccess(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item.Id(2043001), true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 15 stat adds + AddSlots(-1) + AddLevel(1) = 17 changes.
	if len(changes) != 17 {
		t.Errorf("expected 17 changes for regular success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_RegularFailureConsumesSlot(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item.Id(2043001), false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (slot decrement) on plain failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_FailureWithWhiteScrollPreservesSlot(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item.Id(2043001), false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes on white-scroll failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_SpikeSuccess(t *testing.T) {
	ci := makeScrollModel(t, 10, 0, 0)
	equip := createTestEquipableAsset(1072000, 5, 0)
	changes, err := buildScrollChanges(ci, equip, item.ScrollForSpikesOnShoesTenPercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (SetSpike) for spike success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_SpikeFailureNoSlotLoss(t *testing.T) {
	ci := makeScrollModel(t, 10, 0, 0)
	equip := createTestEquipableAsset(1072000, 5, 0)
	changes, err := buildScrollChanges(ci, equip, item.ScrollForSpikesOnShoesTenPercent, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for spike failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_CleanSlateSuccessAddsSlot(t *testing.T) {
	ci := makeScrollModel(t, 1, 0, 0)
	equip := createTestEquipableAsset(1302000, 0, 2)
	changes, err := buildScrollChanges(ci, equip, item.CleanSlateOnePercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (AddSlots) for clean slate success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_ChaosSuccess(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 0)
	equip := asset.NewBuilder(uuid.New(), 1302000).
		SetId(1).
		SetSlots(7).
		SetStrength(10).
		SetDexterity(10).
		Build()
	changes, err := buildScrollChanges(ci, equip, item.ChaosScrollSixtyPercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 chaos stat changes + AddSlots(-1) + AddLevel(1) = 4.
	if len(changes) != 4 {
		t.Errorf("expected 4 changes for chaos success on 2 non-zero stats, got %d", len(changes))
	}
}

// discardLogger returns a logger for computeEffectPlan tests; the function
// only logs on the morphRandom roll-failure path.
func discardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// extractConsumable builds a consumable model the same way production data
// arrives: a RestModel literal run through the public Extract (design §4.4).
func extractConsumable(t *testing.T, rm consumable3.RestModel) consumable3.Model {
	t.Helper()
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

// T8: refactor regression — representative pre-existing items produce the same
// decisions ApplyItemEffects made before the extraction.
func TestComputeEffectPlan_CurePotWithHp(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(500).SetMaxMp(500).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypePoison: 1,
			consumable3.SpecTypeHP:     300,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []string{"POISON"}, plan.cureTypes)
	assert.Equal(t, []int16{300}, plan.hpChanges)
	assert.Empty(t, plan.mpChanges)
	assert.Empty(t, plan.statups)
	assert.Equal(t, int32(0), plan.duration)
}

func TestComputeEffectPlan_StatPotWithTime(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(500).SetMaxMp(500).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeWeaponAttack: 12,
			consumable3.SpecTypeTime:         300000,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Empty(t, plan.cureTypes)
	assert.Empty(t, plan.hpChanges)
	assert.Empty(t, plan.mpChanges)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeWeaponAttack, Amount: 12}}, plan.statups)
	// duration is the WZ `time` spec in milliseconds, passed to atlas-buffs
	// as-is (atlas-buffs schedules expiry as now + duration*time.Millisecond).
	assert.Equal(t, int32(300000), plan.duration)
}

func TestComputeEffectPlan_HpRecoveryPercent(t *testing.T) {
	// Pins the MaxHp * pct floor math: floor(1547 * 0.60) = 928.
	c := character.NewModelBuilder().SetMaxHp(1547).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeHPRecovery: 60,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []int16{928}, plan.hpChanges)
}

// T5 (FR-3 + hp-alongside): fixed-morph 221 item applies MORPH statup with the
// morph id, duration = the WZ time spec (ms), and the coexisting hp spec still heals.
func TestComputeEffectPlan_FixedMorphWithHp(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(100).Build()
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeMorph: 2,
			consumable3.SpecTypeTime:  600000,
			consumable3.SpecTypeHP:    50,
		},
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeMorph, Amount: 2}}, plan.statups)
	assert.Equal(t, int32(600000), plan.duration)
	assert.Equal(t, []int16{50}, plan.hpChanges)
}

// T6: 2211000-shaped item — no fixed morph spec, non-empty morphRandom table.
// Exactly one MORPH statup whose amount is a table key; hp still applies.
func TestComputeEffectPlan_RandomMorphOnly(t *testing.T) {
	c := character.NewModelBuilder().SetMaxHp(100).Build()
	morphs := map[uint32]uint32{20: 50, 21: 30, 22: 20}
	ci := extractConsumable(t, consumable3.RestModel{
		Spec: map[consumable3.SpecType]int32{
			consumable3.SpecTypeTime: 600000,
			consumable3.SpecTypeHP:   50,
		},
		Morphs: morphs,
	})
	plan := computeEffectPlan(discardLogger(), c, ci)
	if assert.Len(t, plan.statups, 1) {
		s := plan.statups[0]
		assert.Equal(t, ts.TemporaryStatTypeMorph, s.Type)
		_, present := morphs[uint32(s.Amount)]
		assert.True(t, present, "morph amount %d is not a table key", s.Amount)
	}
	assert.Equal(t, []int16{50}, plan.hpChanges)
	assert.Equal(t, int32(600000), plan.duration)
}

// T7 (FR-7): fixed morph wins over a table that deliberately does not contain it.
func TestComputeEffectPlan_FixedMorphPrecedence(t *testing.T) {
	ci := extractConsumable(t, consumable3.RestModel{
		Spec:   map[consumable3.SpecType]int32{consumable3.SpecTypeMorph: 2},
		Morphs: map[uint32]uint32{20: 100},
	})
	plan := computeEffectPlan(discardLogger(), character.NewModelBuilder().Build(), ci)
	assert.Equal(t, []stat.Model{{Type: ts.TemporaryStatTypeMorph, Amount: 2}}, plan.statups)
}

// Design §6: an unusable (zero-total) table skips only the morph statup;
// other specs still apply.
func TestComputeEffectPlan_ZeroWeightMorphTableSkipsMorphOnly(t *testing.T) {
	ci := extractConsumable(t, consumable3.RestModel{
		Spec:   map[consumable3.SpecType]int32{consumable3.SpecTypeHP: 50},
		Morphs: map[uint32]uint32{20: 0},
	})
	plan := computeEffectPlan(discardLogger(), character.NewModelBuilder().Build(), ci)
	assert.Empty(t, plan.statups)
	assert.Equal(t, []int16{50}, plan.hpChanges)
}

func TestPetSkillPouchClassification(t *testing.T) {
	// 0519 items route to the pet-skill branch, not the standard consumer.
	for _, id := range []item.Id{5190001, 5190006, 5191001} {
		if item.GetClassification(id) != item.ClassificationPetSkill {
			t.Errorf("GetClassification(%d) = %d, want 519", id, item.GetClassification(id))
		}
		if usesStandardConsumer(id) {
			t.Errorf("usesStandardConsumer(%d) = true, want false", id)
		}
	}
}
