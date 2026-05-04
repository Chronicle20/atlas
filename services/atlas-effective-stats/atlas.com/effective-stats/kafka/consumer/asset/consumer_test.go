package asset

import (
	"strconv"
	"strings"
	"testing"

	"atlas-effective-stats/character"
	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/kafka/message/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/sirupsen/logrus"
)

// TestHandleAssetMoved_CapeUnlocksWeapon_CrossAssetQualification reproduces
// PRD §4.2 (cross-asset qualification): a weapon gated on STR 110 stays
// unqualified until a cape that grants +10 STR is equipped, at which point
// the weapon's bonuses must enter the qualifying snapshot.
//
// The test drives the full equip path end-to-end through handleAssetMoved
// (which dispatches to handleItemEquipped), exercising the same code path
// production hits when atlas-inventory emits an asset MOVED event.
func TestHandleAssetMoved_CapeUnlocksWeapon_CrossAssetQualification(t *testing.T) {
	setupAssetTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	cfg := stubConfig{
		character: stubCharacter{
			level: 30, jobId: 100,
			str: 100, dex: 4, intl: 4, luk: 4, maxHp: 1430, maxMp: 1000,
		},
		equipped: []stubEquipped{
			{assetId: 1, templateId: 1402000, slot: -11, wAtk: 50},
		},
		equipmentReqs: map[uint32]equipmentReqs{
			1402000: {reqStr: 110},
			1102000: {reqStr: 0},
		},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	if _, _, err := character.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345); err != nil {
		t.Fatalf("GetEffectiveStats: %v", err)
	}
	m, _ := character.GetRegistry().Get(ctx, 12345)
	if hasEquipmentBonus(m, 1) {
		t.Fatalf("pre: weapon should be unqualified (base STR 100 < 110)")
	}

	// Now equip the cape: append it to the inventory snapshot and replace the
	// stub set so the handler's RequestEquipCompartment call sees both items.
	cfg.equipped = append(cfg.equipped, stubEquipped{
		assetId: 2, templateId: 1102000, slot: -9, str: 10,
	})
	stubs.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleAssetMoved(l, ctx, asset.StatusEvent[asset.MovedStatusEventBody]{
		CharacterId: 12345,
		AssetId:     2,
		TemplateId:  1102000,
		Slot:        -9,
		Type:        asset.StatusEventTypeMoved,
		Body:        asset.MovedStatusEventBody{OldSlot: 100},
	})

	m, _ = character.GetRegistry().Get(ctx, 12345)
	if !hasEquipmentBonus(m, 1) {
		t.Errorf("post-equip: weapon should now qualify (effective STR 100+10=110 >= 110)")
	}
	if !hasEquipmentBonus(m, 2) {
		t.Errorf("post-equip: cape should be present in qualifying set")
	}
}

// TestHandleItemEquipped_AssetNotInCompartment verifies the diagnostic WARN
// path that fires when an equip MOVED event references an asset id that the
// equip compartment fetch doesn't return. Likely cause in production: a
// read-after-write race or a concurrent delete between event emission and the
// compartment GET.
func TestHandleItemEquipped_AssetNotInCompartment(t *testing.T) {
	setupAssetTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	cfg := stubConfig{
		character: stubCharacter{level: 30, jobId: 100, str: 100, dex: 4, intl: 4, luk: 4, maxHp: 1430, maxMp: 1000},
		// Compartment intentionally has only assetId=1; the equip event below
		// targets assetId=999 which is absent.
		equipped: []stubEquipped{
			{assetId: 1, templateId: 1402000, slot: -11, wAtk: 50},
		},
		equipmentReqs: map[uint32]equipmentReqs{1402000: {}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _, hook := createTestContextWithHook()
	handleAssetMoved(l, ctx, asset.StatusEvent[asset.MovedStatusEventBody]{
		CharacterId: 12345,
		AssetId:     999,
		TemplateId:  1402000,
		Slot:        -11,
		Type:        asset.StatusEventTypeMoved,
		Body:        asset.MovedStatusEventBody{OldSlot: 7},
	})

	m, _ := character.GetRegistry().Get(ctx, 12345)
	if hasEquipmentBonus(m, 999) {
		t.Errorf("absent asset must not enter qualifying snapshot")
	}

	entry := findWarnContaining(hook, "not present in the equip compartment")
	if entry == nil {
		t.Fatalf("expected a WARN about absent asset; got entries: %v", hook.AllEntries())
	}
	msg := entry.Message
	for _, want := range []string{"asset [999]", "character [12345]", "slot=[-11]", "oldSlot=[7]"} {
		if !strings.Contains(msg, want) {
			t.Errorf("WARN missing %q; got %q", want, msg)
		}
	}
}

// TestHandleItemEquipped_AssetAtNonEquippedSlot verifies the diagnostic WARN
// path that fires when an equip MOVED event references an asset that the
// compartment fetch returns at a positive (inventory) slot. Post-PR #395 this
// should never happen; if it does, the WARN must surface the slot mismatch so
// an operator can spot upstream slot-semantic inversion at a glance.
func TestHandleItemEquipped_AssetAtNonEquippedSlot(t *testing.T) {
	setupAssetTest(t)
	t.Cleanup(equipment.ResetCacheForTest)

	cfg := stubConfig{
		character: stubCharacter{level: 30, jobId: 100, str: 100, dex: 4, intl: 4, luk: 4, maxHp: 1430, maxMp: 1000},
		// The asset exists in the compartment, but at a positive slot — so
		// GetEquipableData() returns ok=false and equipData stays nil while
		// foundInCompart=true. This is the case-2 diagnostic branch.
		equipped: []stubEquipped{
			{assetId: 42, templateId: 1402000, slot: 3, wAtk: 50},
		},
		equipmentReqs: map[uint32]equipmentReqs{1402000: {}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _, hook := createTestContextWithHook()
	handleAssetMoved(l, ctx, asset.StatusEvent[asset.MovedStatusEventBody]{
		CharacterId: 12345,
		AssetId:     42,
		TemplateId:  1402000,
		Slot:        -11,
		Type:        asset.StatusEventTypeMoved,
		Body:        asset.MovedStatusEventBody{OldSlot: 3},
	})

	m, _ := character.GetRegistry().Get(ctx, 12345)
	if hasEquipmentBonus(m, 42) {
		t.Errorf("non-equipped asset must not enter qualifying snapshot")
	}

	entry := findWarnContaining(hook, "non-equipped slot")
	if entry == nil {
		t.Fatalf("expected a WARN about non-equipped slot; got entries: %v", hook.AllEntries())
	}
	msg := entry.Message
	for _, want := range []string{"asset [42]", "character [12345]", "non-equipped slot [3]", "slot=[-11]", "oldSlot=[3]", "slot-semantic inversion"} {
		if !strings.Contains(msg, want) {
			t.Errorf("WARN missing %q; got %q", want, msg)
		}
	}
}

// findWarnContaining returns the first WARN entry whose message contains
// substr, or nil if none. Iteration walks AllEntries() so any prior log noise
// from initializer paths doesn't shadow the diagnostic we care about.
func findWarnContaining(hook interface{ AllEntries() []*logrus.Entry }, substr string) *logrus.Entry {
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && strings.Contains(e.Message, substr) {
			return e
		}
	}
	return nil
}

// hasEquipmentBonus reports whether the model's qualifying-snapshot bonus
// list contains any entry sourced from the given equipment asset id. The
// "equipment:<id>" source string is set by extractEquipmentBonuses in
// consumer.go and the matching extractor in character/initializer.go.
func hasEquipmentBonus(m character.Model, assetId uint32) bool {
	src := "equipment:" + strconv.FormatUint(uint64(assetId), 10)
	for _, b := range m.Bonuses() {
		if b.Source() == src {
			return true
		}
	}
	return false
}
