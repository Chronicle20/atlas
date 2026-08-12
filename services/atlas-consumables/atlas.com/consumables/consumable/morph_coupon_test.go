package consumable

import (
	"atlas-consumables/cash"
	"atlas-consumables/character/buff/stat"
	"testing"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
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
