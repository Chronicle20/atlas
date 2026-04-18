package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/data/skill/effect"
	"testing"
	"time"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

// Reference item IDs used across the tests.
const (
	arrowForBow         uint32 = 2060000 // Arrow for Bow (classification 206)
	arrowForBowOther    uint32 = 2060001 // Arrow for Bow (different template, still classification 206)
	arrowForCrossbow    uint32 = 2061000 // Bolt for Crossbow (classification 206)
	throwingStarSubi    uint32 = 2070000 // Subi Throwing Stars (classification 207)
	bulletAdvanced      uint32 = 2330000 // Advanced Bullet (classification 233)
	nonProjectilePotion uint32 = 2000000 // Red Potion (classification 200)
)

func makeAsset(slot int16, templateId uint32, qty uint32) asset.Model {
	return asset.NewModelBuilder(1, uuid.New(), templateId).
		SetSlot(slot).
		SetQuantity(qty).
		MustBuild()
}

func TestRequiredClassification(t *testing.T) {
	cases := []struct {
		name   string
		w      item.WeaponType
		wantC  item.Classification
		wantOk bool
	}{
		{"bow", item.WeaponTypeBow, item.ClassificationConsumableArrow, true},
		{"crossbow", item.WeaponTypeCrossbow, item.ClassificationConsumableArrow, true},
		{"claw", item.WeaponTypeClaw, item.ClassificationConsumableThrowingStar, true},
		{"gun", item.WeaponTypeGun, item.ClassificationBullet, true},
		{"sword non-ranged", item.WeaponTypeOneHandedSword, 0, false},
		{"none", item.WeaponTypeNone, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := requiredClassification(tc.w)
			if ok != tc.wantOk || c != tc.wantC {
				t.Fatalf("got (%d, %v), want (%d, %v)", c, ok, tc.wantC, tc.wantOk)
			}
		})
	}
}

func buffWithStat(statType ts.TemporaryStatType) buff.Model {
	future := time.Now().Add(time.Minute)
	return buff.NewBuff(0, 1, 0, []stat.Model{stat.NewStat(string(statType), 1)}, time.Now(), future)
}

func expiredBuffWithStat(statType ts.TemporaryStatType) buff.Model {
	past := time.Now().Add(-time.Minute)
	return buff.NewBuff(0, 1, 0, []stat.Model{stat.NewStat(string(statType), 1)}, past, past)
}

func TestComputeCount(t *testing.T) {
	se := effect.Model{} // BulletConsume == 0 → base 1.
	if got := computeCount(item.WeaponTypeBow, se, nil); got != 1 {
		t.Fatalf("bow base count = %d, want 1", got)
	}
	if got := computeCount(item.WeaponTypeClaw, se, nil); got != 1 {
		t.Fatalf("claw base count no buff = %d, want 1", got)
	}
	// Shadow Partner doubles claw count.
	buffs := []buff.Model{buffWithStat(ts.TemporaryStatTypeShadowPartner)}
	if got := computeCount(item.WeaponTypeClaw, se, buffs); got != 2 {
		t.Fatalf("claw + SP count = %d, want 2", got)
	}
	// Shadow Partner does not affect bow/gun/crossbow.
	if got := computeCount(item.WeaponTypeBow, se, buffs); got != 1 {
		t.Fatalf("bow + SP count = %d, want 1", got)
	}
	// Expired Shadow Partner should not double.
	expired := []buff.Model{expiredBuffWithStat(ts.TemporaryStatTypeShadowPartner)}
	if got := computeCount(item.WeaponTypeClaw, se, expired); got != 1 {
		t.Fatalf("claw + expired SP count = %d, want 1", got)
	}
}

func TestResolvePlan_SingleSlotPreferred(t *testing.T) {
	assets := []asset.Model{
		makeAsset(1, arrowForBow, 100),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 1, 1)
	if len(draws) != 1 || draws[0].Slot != 1 || draws[0].Quantity != 1 {
		t.Fatalf("unexpected draws: %+v", draws)
	}
	if available != 1 {
		t.Fatalf("available = %d, want 1", available)
	}
}

func TestResolvePlan_ClientHintMissFallbackToOther(t *testing.T) {
	// Client says slot 3 (which has 1 arrow, not enough for count=4),
	// but slot 2 has a full stack. Fallback should pick slot 2 as a single draw.
	assets := []asset.Model{
		makeAsset(2, arrowForBow, 100),
		makeAsset(3, arrowForBowOther, 1),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 3, 4)
	if len(draws) != 1 || draws[0].Slot != 2 || draws[0].Quantity != 4 {
		t.Fatalf("unexpected draws: %+v", draws)
	}
	if available != 4 {
		t.Fatalf("available = %d, want 4", available)
	}
}

func TestResolvePlan_MultiSlotDraw(t *testing.T) {
	// No single slot has 5, but combined slots 1+2 cover it. Ascending order.
	assets := []asset.Model{
		makeAsset(2, arrowForBowOther, 3),
		makeAsset(1, arrowForBow, 3),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 0, 5)
	if len(draws) != 2 {
		t.Fatalf("expected 2 draws, got %+v", draws)
	}
	if draws[0].Slot != 1 || draws[0].Quantity != 3 {
		t.Fatalf("draw 0 = %+v, want slot 1 qty 3", draws[0])
	}
	if draws[1].Slot != 2 || draws[1].Quantity != 2 {
		t.Fatalf("draw 1 = %+v, want slot 2 qty 2", draws[1])
	}
	if available != 5 {
		t.Fatalf("available = %d, want 5", available)
	}
}

func TestResolvePlan_TotalShortfall(t *testing.T) {
	// Need 10, only 4 available across both slots: consume everything; flag shortfall.
	assets := []asset.Model{
		makeAsset(1, arrowForBow, 3),
		makeAsset(2, arrowForBowOther, 1),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 0, 10)
	if len(draws) != 2 {
		t.Fatalf("expected 2 draws, got %+v", draws)
	}
	if available != 4 {
		t.Fatalf("available = %d, want 4", available)
	}
}

func TestResolvePlan_NoMatchingSlot(t *testing.T) {
	assets := []asset.Model{
		makeAsset(1, nonProjectilePotion, 10),
		makeAsset(2, throwingStarSubi, 100),
		makeAsset(3, bulletAdvanced, 100),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 0, 1)
	if len(draws) != 0 {
		t.Fatalf("expected no draws, got %+v", draws)
	}
	if available != 0 {
		t.Fatalf("available = %d, want 0", available)
	}
}

func TestResolvePlan_EmptyQtySlotsSkipped(t *testing.T) {
	// A rechargeable stack sitting at qty=0 should not be drawn from.
	assets := []asset.Model{
		makeAsset(1, throwingStarSubi, 0),
		makeAsset(2, throwingStarSubi, 5),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableThrowingStar, 1, 3)
	if len(draws) != 1 || draws[0].Slot != 2 || draws[0].Quantity != 3 {
		t.Fatalf("unexpected draws: %+v", draws)
	}
	if available != 3 {
		t.Fatalf("available = %d, want 3", available)
	}
}

func TestResolvePlan_ClientHintExactMatch(t *testing.T) {
	// Client-suggested slot has exactly enough; preferred over equally-valid slot 1.
	assets := []asset.Model{
		makeAsset(1, arrowForBow, 100),
		makeAsset(2, arrowForBowOther, 10),
	}
	draws, available := resolvePlan(assets, item.ClassificationConsumableArrow, 2, 5)
	if len(draws) != 1 || draws[0].Slot != 2 || draws[0].Quantity != 5 {
		t.Fatalf("unexpected draws: %+v", draws)
	}
	if available != 5 {
		t.Fatalf("available = %d, want 5", available)
	}
}
