package consumable

import (
	"atlas-consumables/cash"
	"atlas-consumables/character/buff/stat"
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	cashmock "atlas-consumables/cash/mock"
	buffmock "atlas-consumables/character/buff/mock"
	charmock "atlas-consumables/character/mock"
	compmock "atlas-consumables/compartment/mock"
	mapcharmock "atlas-consumables/map/character/mock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	fieldc "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map2 "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// extractCash builds a cash.Model through the package's own exported Extract,
// so no test-only constructor is introduced (CLAUDE.md test-helper rule).
func extractCash(t *testing.T, spec map[cash.SpecType]int32) cash.Model {
	t.Helper()
	m, err := cash.Extract(cash.RestModel{Id: 5300000, SlotMax: 200, Spec: spec})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// TestComputeMorphCouponPlan covers FR-3.5 and every FR-3.7 permutation. The
// full-spec row uses the real WZ values for 5300000 (morph 1, hp 50,
// time 600000 ms), verified against two local Item.wz/Cash/0530.img.xml copies.
func TestComputeMorphCouponPlan(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[cash.SpecType]int32
		wantHp       int16
		wantMorph    int32 // 0 = expect no morph statup
		wantDuration int32
	}{
		{
			name:         "full spec (5300000)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    1,
			wantDuration: 600000,
		},
		{
			name:         "morph 3 (5300002)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 3, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    3,
			wantDuration: 600000,
		},
		{
			name:         "morph absent: heals, does not morph",
			spec:         map[cash.SpecType]int32{cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "morph zero: heals, does not morph",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 0, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "hp absent: morphs, does not heal",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 2, cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    2,
			wantDuration: 600000,
		},
		{
			name:         "hp zero: morphs, does not heal",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 2, cash.SpecTypeHp: 0, cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    2,
			wantDuration: 600000,
		},
		{
			name:         "both absent: does nothing (stale ingest)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "empty spec: does nothing, duration zero",
			spec:         map[cash.SpecType]int32{},
			wantHp:       0,
			wantMorph:    0,
			wantDuration: 0,
		},
		{
			name:         "time absent: morph applied with zero duration",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50},
			wantHp:       50,
			wantMorph:    1,
			wantDuration: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := computeMorphCouponPlan(extractCash(t, tc.spec))

			if plan.hp != tc.wantHp {
				t.Errorf("hp = %d, want %d", plan.hp, tc.wantHp)
			}
			// FR-3.6: the WZ `time` value is passed through unscaled — atlas-buffs
			// expects milliseconds. Any *1000 or /1000 fails here.
			if plan.duration != tc.wantDuration {
				t.Errorf("duration = %d, want %d (raw WZ ms, unscaled)", plan.duration, tc.wantDuration)
			}
			if tc.wantMorph == 0 {
				if len(plan.statups) != 0 {
					t.Fatalf("statups = %+v, want none", plan.statups)
				}
				return
			}
			if len(plan.statups) != 1 {
				t.Fatalf("len(statups) = %d, want 1", len(plan.statups))
			}
			want := stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: tc.wantMorph}
			if plan.statups[0] != want {
				t.Errorf("statups[0] = %+v, want %+v", plan.statups[0], want)
			}
		})
	}
}

// TestRoutesToMorphCoupon pins FR-1.3 / FR-3.2's gate: selection is by item
// classification, never by the cash-slot type byte. The negatives are the
// classifications whose type bytes collide with 530's across versions
// (gachapon 522 -> 40 on GMS>=95; pet evolution 538 -> 41 on GMS<95) plus the
// use-tab transformation potion (221), which must keep routing to
// ConsumeStandard.
func TestRoutesToMorphCoupon(t *testing.T) {
	for _, id := range []item2.Id{5300000, 5300001, 5300002} {
		if item2.GetClassification(id) != item2.ClassificationTransformationCoupon {
			t.Fatalf("fixture invalid: GetClassification(%d) = %d, want 530", id, item2.GetClassification(id))
		}
		if !routesToMorphCoupon(id) {
			t.Errorf("routesToMorphCoupon(%d) = false, want true", id)
		}
	}
	for _, id := range []item2.Id{
		5220000, // 522 gachapon coupon  -> cash-slot type 40 on GMS >= 95
		5380000, // 538 pet evolution    -> cash-slot type 41 on GMS <  95
		5211000, // 521 EXP coupon
		2210000, // 221 use-tab transformation potion -> ConsumeStandard
		2000000, // 200 HP potion
	} {
		if routesToMorphCoupon(id) {
			t.Errorf("routesToMorphCoupon(%d) = true, want false (classification %d)", id, item2.GetClassification(id))
		}
	}
}

// TestMorphCouponNotStandardConsumer pins FR-3.2's negative half: ConsumeStandard
// hard-codes inventory2.TypeValueUse and fetches from the *consumable* data
// resource, where 5300000 does not exist. A 530 item must never reach it.
func TestMorphCouponNotStandardConsumer(t *testing.T) {
	for _, id := range []item2.Id{5300000, 5300001, 5300002} {
		if usesStandardConsumer(id) {
			t.Errorf("usesStandardConsumer(%d) = true, want false", id)
		}
	}
}

// consumeItemCall records one compartment.ConsumeItem invocation so the
// compartment-type contract (FR-3.4) can be asserted, not assumed.
type consumeItemCall struct {
	characterId   uint32
	inventoryType inventory2.Type
	slot          int16
}

type applyCall struct {
	fromId   uint32
	sourceId int32
	level    byte
	duration int32
	statups  []stat.Model
}

type changeHPCall struct {
	characterId uint32
	amount      int16
}

// morphCouponHarness wires the five package mocks into a morphCouponDeps and
// captures every outbound call. onError records the ConsumeError route without
// needing a live Kafka broker.
type morphCouponHarness struct {
	deps         morphCouponDeps
	consumeItems []consumeItemCall
	applies      []applyCall
	hpChanges    []changeHPCall
	errors       []error
}

func newMorphCouponHarness(t *testing.T, ci cash.Model, cashErr error) *morphCouponHarness {
	t.Helper()
	f := fieldc.NewBuilder(world.Id(0), channel.Id(0), _map2.Id(100000000)).Build()
	h := &morphCouponHarness{}
	h.deps = morphCouponDeps{
		cash: &cashmock.ProcessorMock{
			GetByIdFunc: func(uint32) (cash.Model, error) { return ci, cashErr },
		},
		fields: &mapcharmock.ProcessorMock{
			GetMapFunc: func(uint32) (fieldc.Model, error) { return f, nil },
		},
		compartment: &compmock.ProcessorMock{
			ConsumeItemFunc: func(characterId uint32, it inventory2.Type, _ uuid.UUID, slot int16) error {
				h.consumeItems = append(h.consumeItems, consumeItemCall{characterId, it, slot})
				return nil
			},
		},
		character: &charmock.ProcessorMock{
			ChangeHPFunc: func(_ fieldc.Model, characterId uint32, amount int16) error {
				h.hpChanges = append(h.hpChanges, changeHPCall{characterId, amount})
				return nil
			},
		},
		buff: &buffmock.ProcessorMock{
			ApplyFunc: func(_ fieldc.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32] {
				h.applies = append(h.applies, applyCall{fromId, sourceId, level, duration, statups})
				return func(uint32) error { return nil }
			},
		},
		onError: func(err error) error {
			h.errors = append(h.errors, err)
			return err
		},
	}
	return h
}

func fullMorphSpec() map[cash.SpecType]int32 {
	return map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000}
}

// TestConsumeMorphCouponSuccess pins FR-3.4, FR-3.5 and FR-3.6 in one pass:
// the coupon leaves the CASH compartment, the HP heal is issued, and the morph
// statup carries source = -itemId, level 0, and the raw WZ duration in ms.
func TestConsumeMorphCouponSuccess(t *testing.T) {
	const characterId = uint32(555)
	const slot = int16(3)
	const itemId = item2.Id(5300000)

	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)

	if err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), characterId, slot, itemId); err != nil {
		t.Fatalf("consumeMorphCoupon: %v", err)
	}

	if len(h.errors) != 0 {
		t.Fatalf("errors = %v, want none", h.errors)
	}
	if len(h.consumeItems) != 1 {
		t.Fatalf("ConsumeItem call count = %d, want 1", len(h.consumeItems))
	}
	// FR-3.4: the CASH compartment, not the Use compartment. ConsumeStandard's
	// hard-coded TypeValueUse is exactly why this consumer exists.
	if got := h.consumeItems[0].inventoryType; got != inventory2.TypeValueCash {
		t.Errorf("ConsumeItem inventoryType = %d, want TypeValueCash (%d)", got, inventory2.TypeValueCash)
	}
	if h.consumeItems[0].characterId != characterId || h.consumeItems[0].slot != slot {
		t.Errorf("ConsumeItem = %+v, want characterId %d slot %d", h.consumeItems[0], characterId, slot)
	}

	if len(h.hpChanges) != 1 || h.hpChanges[0].amount != 50 {
		t.Fatalf("hpChanges = %+v, want one call of amount 50", h.hpChanges)
	}

	if len(h.applies) != 1 {
		t.Fatalf("Apply call count = %d, want 1", len(h.applies))
	}
	a := h.applies[0]
	if a.sourceId != -int32(itemId) {
		t.Errorf("Apply sourceId = %d, want %d (-itemId)", a.sourceId, -int32(itemId))
	}
	if a.level != 0 {
		t.Errorf("Apply level = %d, want 0", a.level)
	}
	// FR-3.6: 600000 is the WZ value in milliseconds, unscaled.
	if a.duration != 600000 {
		t.Errorf("Apply duration = %d, want 600000 (raw WZ ms)", a.duration)
	}
	want := stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: 1}
	if len(a.statups) != 1 || a.statups[0] != want {
		t.Errorf("Apply statups = %+v, want [%+v]", a.statups, want)
	}
	if a.fromId != characterId {
		t.Errorf("Apply fromId = %d, want %d", a.fromId, characterId)
	}
}

// TestConsumeMorphCouponCashFetchFailureKeepsCoupon pins FR-3.3: every fallible
// read happens BEFORE the commit, so a data failure releases the reservation via
// ConsumeError and the player keeps the paid item.
func TestConsumeMorphCouponCashFetchFailureKeepsCoupon(t *testing.T) {
	wantErr := errors.New("cash-items 404")
	h := newMorphCouponHarness(t, cash.Model{}, wantErr)

	err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(h.errors) != 1 {
		t.Fatalf("onError call count = %d, want 1", len(h.errors))
	}
	if len(h.consumeItems) != 0 {
		t.Errorf("ConsumeItem call count = %d, want 0 — the coupon must stay in the inventory", len(h.consumeItems))
	}
	if len(h.applies) != 0 || len(h.hpChanges) != 0 {
		t.Errorf("effects applied on a failed read: applies %+v, hp %+v", h.applies, h.hpChanges)
	}
}

// TestConsumeMorphCouponConsumeFailureAppliesNoEffects: a failed commit must not
// morph or heal — the reservation is released and nothing else happens.
func TestConsumeMorphCouponConsumeFailureAppliesNoEffects(t *testing.T) {
	wantErr := errors.New("reservation expired")
	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)
	h.deps.compartment = &compmock.ProcessorMock{
		ConsumeItemFunc: func(uint32, inventory2.Type, uuid.UUID, int16) error { return wantErr },
	}

	err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(h.applies) != 0 || len(h.hpChanges) != 0 {
		t.Errorf("effects applied after a failed commit: applies %+v, hp %+v", h.applies, h.hpChanges)
	}
}

// TestConsumeMorphCouponZeroSpecs pins FR-3.7 through the consumer, not just the
// planner: each half is independently skippable and neither blocks the consume.
func TestConsumeMorphCouponZeroSpecs(t *testing.T) {
	tests := []struct {
		name        string
		spec        map[cash.SpecType]int32
		wantApplies int
		wantHp      int
	}{
		{"morph absent", map[cash.SpecType]int32{cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000}, 0, 1},
		{"hp absent", map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeTime: 600000}, 1, 0},
		{"both absent (stale ingest)", map[cash.SpecType]int32{cash.SpecTypeTime: 600000}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMorphCouponHarness(t, extractCash(t, tc.spec), nil)
			if err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000); err != nil {
				t.Fatalf("consumeMorphCoupon: %v", err)
			}
			// The coupon is consumed in every row — an empty spec is not an error.
			if len(h.consumeItems) != 1 {
				t.Errorf("ConsumeItem call count = %d, want 1", len(h.consumeItems))
			}
			if len(h.applies) != tc.wantApplies {
				t.Errorf("Apply call count = %d, want %d", len(h.applies), tc.wantApplies)
			}
			if len(h.hpChanges) != tc.wantHp {
				t.Errorf("ChangeHP call count = %d, want %d", len(h.hpChanges), tc.wantHp)
			}
			if len(h.errors) != 0 {
				t.Errorf("errors = %v, want none — an absent spec is not a failure", h.errors)
			}
		})
	}
}

// TestConsumeMorphCouponReuseWhileMorphedApplies pins FR-3.8: using a second
// coupon while already transformed issues a second Apply unconditionally. There
// is no "already morphed" rejection branch — replace-and-restart is the default
// overwrite behaviour of the atlas-buffs apply path. This test asserts the
// ABSENCE of a rejection, so adding one would fail here.
func TestConsumeMorphCouponReuseWhileMorphedApplies(t *testing.T) {
	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)
	l := logrus.New()

	for i := 0; i < 2; i++ {
		if err := consumeMorphCoupon(l, context.Background(), h.deps, uuid.New(), 555, 3, 5300000); err != nil {
			t.Fatalf("use %d: %v", i+1, err)
		}
	}

	if len(h.applies) != 2 {
		t.Fatalf("Apply call count = %d, want 2 (second use replaces the morph and restarts the timer)", len(h.applies))
	}
	if !reflect.DeepEqual(h.applies[0], h.applies[1]) {
		t.Errorf("second Apply differs from the first: %+v vs %+v", h.applies[0], h.applies[1])
	}
	if len(h.consumeItems) != 2 {
		t.Errorf("ConsumeItem call count = %d, want 2", len(h.consumeItems))
	}
}

// TestConsumeMorphCouponBindsRealProcessors: the exported wrapper must build a
// complete deps set. A nil collaborator would panic at runtime on the first
// coupon use, which no other test in this file would catch.
func TestConsumeMorphCouponBindsRealProcessors(t *testing.T) {
	if ConsumeMorphCoupon(uuid.New(), 555, 3, 5300000) == nil {
		t.Fatal("ConsumeMorphCoupon returned a nil ItemConsumer")
	}
}

// TestMorphCouponRoutedBeforeRewardFallback guards the branch ORDER in
// RequestItemConsume. The reward-table fallback queries the consumable data
// resource, which has no cash items, so a 530 item reaching it falls through to
// ConsumeBare — the coupon is destroyed and nothing is applied. A source check
// is the honest test here: the routing chain is a private if/else inside a
// method that opens a Kafka reservation, so it has no seam to observe.
func TestMorphCouponRoutedBeforeRewardFallback(t *testing.T) {
	src, err := os.ReadFile("processor.go")
	if err != nil {
		t.Fatalf("read processor.go: %v", err)
	}
	morphAt := bytes.Index(src, []byte("routesToMorphCoupon(itemId)"))
	if morphAt < 0 {
		t.Fatal("RequestItemConsume has no routesToMorphCoupon branch")
	}
	fallbackAt := bytes.Index(src, []byte("validateRewardTable(ci.Rewards())"))
	if fallbackAt < 0 {
		t.Fatal("could not locate the reward-table fallback branch")
	}
	if morphAt > fallbackAt {
		t.Error("the morph-coupon branch must precede the reward-table fallback, or 530 items fall through to ConsumeBare")
	}
}
