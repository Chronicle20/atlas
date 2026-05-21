package workers

import "testing"

// TestImgID pins the suffix-tolerance contract that the per-id icon and map
// emit loops depend on. The wz library's parseDirectory strips ".img" from
// stored image names (directory.go:127), so img.Name() returns just the
// numeric id. An earlier version of imgID required ".img" — meaning every
// `imgID(img.Name())` returned (0, false), every icon/map loop iterated
// `continue`, and scanned=0 across every worker on PR-544.
func TestImgID(t *testing.T) {
	cases := []struct {
		name    string
		wantId  uint32
		wantOk  bool
	}{
		// Names as the wz library actually produces them (no .img suffix).
		// These are the SHIPPING-DATA cases — must work.
		{"0100100", 100100, true},
		{"00100100", 100100, true},
		{"0", 0, true},
		{"100000000", 100000000, true},
		// Names with the .img suffix kept on (e.g. XML-walker paths). Tolerated.
		{"0100100.img", 100100, true},
		// Non-numeric names that share Mob.wz/Skill.wz/etc. with id-named
		// images — e.g. MobSkill.img, BFSkill, AreaCode. Must reject.
		{"MobSkill", 0, false},
		{"BFSkill", 0, false},
		{"AreaCode", 0, false},
		{"", 0, false},
		// Overflow guard: ParseUint with bitSize=32 rejects values > 2^32-1.
		{"99999999999", 0, false},
	}
	for _, tc := range cases {
		gotId, gotOk := imgID(tc.name)
		if gotOk != tc.wantOk || gotId != tc.wantId {
			t.Errorf("imgID(%q) = (%d, %v), want (%d, %v)", tc.name, gotId, gotOk, tc.wantId, tc.wantOk)
		}
	}
}
