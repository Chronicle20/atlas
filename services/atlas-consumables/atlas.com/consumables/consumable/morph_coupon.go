package consumable

import (
	"atlas-consumables/cash"
	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/compartment"
	character2 "atlas-consumables/map/character"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// morphCouponDeps is the collaborator set consumeMorphCoupon needs.
//
// Split out from the exported consumer so the ordering contract — every fallible
// read BEFORE the commit (FR-3.3), the CASH compartment on every path (FR-3.4),
// the unscaled millisecond duration (FR-3.6) — is pinnable with the package's
// existing mocks. ConsumeMorphCoupon below binds the real processors.
type morphCouponDeps struct {
	cash        cash.Processor
	fields      character2.Processor
	compartment compartment.Processor
	character   character.Processor
	buff        buff.Processor
	// onError releases the reservation and notifies the client. Bound to
	// ProcessorImpl.ConsumeError with the CASH compartment already applied.
	onError func(err error) error
}

// consumeMorphCoupon applies a transformation coupon: read, commit, then heal
// and morph.
//
// Ordering is deliberate and mirrors ConsumeStandard. Both reads are fallible
// and both happen before ConsumeItem, so a data failure returns the coupon to
// the player (FR-3.3) — losing a paid cash item to a no-op is the failure this
// ordering exists to prevent. Effect failures AFTER the commit are logged and
// not rolled back, matching the ApplyItemEffects convention.
func consumeMorphCoupon(l logrus.FieldLogger, ctx context.Context, d morphCouponDeps, transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) error {
	pg, _ := model.NewGroup(ctx)
	ff := model.Submit(pg, func() (field.Model, error) { return d.fields.GetMap(characterId) })
	fi := model.Submit(pg, func() (cash.Model, error) { return d.cash.GetById(uint32(itemId)) })
	if err := pg.Wait(); err != nil {
		return d.onError(err)
	}
	f, ci := ff.Get(), fi.Get()

	plan := computeMorphCouponPlan(ci)

	if err := d.compartment.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot); err != nil {
		return d.onError(err)
	}

	if plan.hp > 0 {
		if err := d.character.ChangeHP(f, characterId, plan.hp); err != nil {
			l.WithError(err).Errorf("Character [%d] consumed transformation coupon [%d] but the HP heal of [%d] failed.", characterId, itemId, plan.hp)
		}
	}
	if len(plan.statups) > 0 {
		// Re-use while already morphed is intentionally unguarded: the second
		// apply replaces the active morph and restarts the timer, which is the
		// default overwrite behaviour of the atlas-buffs apply path (FR-3.8).
		if err := d.buff.Apply(f, characterId, -int32(itemId), byte(0), plan.duration, plan.statups)(characterId); err != nil {
			l.WithError(err).Errorf("Character [%d] consumed transformation coupon [%d] but the morph buff apply failed.", characterId, itemId)
		}
	}

	if plan.hp == 0 && len(plan.statups) == 0 {
		l.Warnf("Character [%d] consumed transformation coupon [%d] but its cash data carries neither a morph nor an hp spec; the tenant's cash WZ likely predates the spec/morph and spec/hp parse.", characterId, itemId)
	}
	return nil
}

// ConsumeMorphCoupon commits a reserved transformation coupon (classification
// 530) from the CASH compartment and applies its morph and HP heal.
//
// It cannot reuse ConsumeStandard: that consumer hard-codes
// inventory2.TypeValueUse and fetches from the *consumable* data resource, where
// cash items do not exist.
func ConsumeMorphCoupon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			d := morphCouponDeps{
				cash:        cash.NewProcessor(l, ctx),
				fields:      character2.NewProcessor(l, ctx),
				compartment: compartment.NewProcessor(l, ctx),
				character:   character.NewProcessor(l, ctx),
				buff:        buff.NewProcessor(l, ctx),
				onError: func(err error) error {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
				},
			}
			return consumeMorphCoupon(l, ctx, d, transactionId, characterId, slot, itemId)
		}
	}
}
