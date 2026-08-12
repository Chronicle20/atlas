package consumable

import (
	"atlas-consumables/cash"
	"atlas-consumables/character/buff/stat"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// morphCouponPlan is the pure result of interpreting a transformation coupon's
// specs: everything the consumer will do, decided before any side effect.
// Mirrors effectPlan's rationale (processor.go) — keeping the decision pure is
// what makes the morph/hp permutations pinnable by plain unit tests.
type morphCouponPlan struct {
	hp       int16        // 0 = no HP change
	statups  []stat.Model // at most one: MORPH
	duration int32        // raw WZ `time` spec in MILLISECONDS, unscaled
}

// computeMorphCouponPlan interprets a 0530 cash item's specs with no side
// effects.
//
// morph and hp are independent (FR-3.7): a coupon whose morph is absent or zero
// still heals and still consumes, and vice versa. A coupon with neither — the
// shape served by a tenant whose cash WZ was ingested before this feature landed
// — consumes and does nothing, which is the honest outcome for absent data.
//
// morphRandom is deliberately not consulted: no item in Item.wz/Cash/0530.img.xml
// carries one in any inspected corpus, so the weighted selector in morph.go stays
// unwired on this path.
func computeMorphCouponPlan(ci cash.Model) morphCouponPlan {
	plan := morphCouponPlan{statups: make([]stat.Model, 0, 1)}

	if val, ok := ci.GetSpec(cash.SpecTypeHp); ok && val > 0 {
		plan.hp = int16(val)
	}
	if val, ok := ci.GetSpec(cash.SpecTypeMorph); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	}
	if val, ok := ci.GetSpec(cash.SpecTypeTime); ok && val > 0 {
		// Milliseconds, passed through as-is. atlas-buffs computes expiry as
		// now + duration*time.Millisecond (ApplyCommandBody.Duration), so any
		// seconds<->milliseconds scaling here makes the morph expire ~1000x
		// early or late. tools/buff-duration-guard.sh enforces this.
		plan.duration = val
	}

	return plan
}

// routesToMorphCoupon reports whether itemId is a transformation coupon and
// therefore routes to ConsumeMorphCoupon.
//
// The gate is item CLASSIFICATION, never the cash-slot type byte. Those bytes
// collide across client versions: atlas-channel's GetCashSlotItemType maps
// classification 530 to type 41 on GMS >= 95 and 40 otherwise, while gachapon
// coupons (522) take 40 on GMS >= 95 and pet evolution (538) takes 41 on
// GMS < 95. A type-keyed gate would silently change meaning at a version bump.
func routesToMorphCoupon(itemId item2.Id) bool {
	return item2.GetClassification(itemId) == item2.ClassificationTransformationCoupon
}
