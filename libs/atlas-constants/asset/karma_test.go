package asset

import "testing"

func TestKarmaFlagFor(t *testing.T) {
	testCases := []struct {
		name       string
		templateId uint32
		wantFlag   Flag
		wantOk     bool
	}{
		{"equip zakum helmet", 1002357, FlagKarmaEquip, true},
		{"equip weapon", 1302000, FlagKarmaEquip, true},
		{"consumable mastery book", 2280000, FlagKarmaUse, true},
		{"setup chair", 3010000, FlagKarmaUse, true},
		{"etc material", 4000000, FlagKarmaUse, true},
		{"cash scissors", 5520000, FlagKarmaUse, true},
		{"pet low", 5000000, 0, false},
		{"pet high", 5009999, 0, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotFlag, gotOk := KarmaFlagFor(tc.templateId)
			if gotFlag != tc.wantFlag || gotOk != tc.wantOk {
				t.Fatalf("KarmaFlagFor(%d) = (%#x, %v), want (%#x, %v)", tc.templateId, gotFlag, gotOk, tc.wantFlag, tc.wantOk)
			}
		})
	}
}

// TestKarmaFlagForEquipIsNotSpikes is the FR-4.5 guard: the equip karma bit
// must never be 0x02, which is FlagSpikes on an equip.
func TestKarmaFlagForEquipIsNotSpikes(t *testing.T) {
	f, ok := KarmaFlagFor(1002357)
	if !ok {
		t.Fatal("KarmaFlagFor refused an equip")
	}
	if f == FlagSpikes {
		t.Fatalf("equip karma bit resolved to FlagSpikes (%#x); marking karma would render spikes", f)
	}
}

func TestKarmaEligible(t *testing.T) {
	testCases := []struct {
		name          string
		scissorsKarma int32
		targetKarma   int32
		want          bool
	}{
		{"v83 model: target not karma-applicable", 0, 0, false},
		{"v83 model: target karma-applicable", 0, 1, true},
		{"v95 model: types match", 1, 1, true},
		{"v95 model: types differ", 1, 2, false},
		{"v95 model: types differ reversed", 2, 1, false},
		{"v95 model: ordinary item, scissors typed", 1, 0, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := KarmaEligible(tc.scissorsKarma, tc.targetKarma); got != tc.want {
				t.Fatalf("KarmaEligible(%d, %d) = %v, want %v", tc.scissorsKarma, tc.targetKarma, got, tc.want)
			}
		})
	}
}
