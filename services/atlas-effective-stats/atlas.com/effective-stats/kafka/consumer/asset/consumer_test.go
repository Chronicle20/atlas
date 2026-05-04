package asset

import (
	"strconv"
	"testing"

	"atlas-effective-stats/character"
	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/kafka/message/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
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
