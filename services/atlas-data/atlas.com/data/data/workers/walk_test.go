package workers

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
)

// TestRootImagesYieldsImgIDParseableNames composes the wz iteration contract
// (parseDirectory strips ".img"; in-memory NewParsedImage preserves whatever
// it was given) with imgID's parse semantics. PR-544's "scanned=0 across
// every worker" symptom was exactly this composition failing: real WZ data
// reached the worker with `.img` stripped, imgID rejected names without
// `.img`, every per-id loop hit `continue`. This test fails the moment
// either side regresses; together with TestImgID and the wz package's
// iteration_contract_test, it pins both endpoints AND their composition.
func TestRootImagesYieldsImgIDParseableNames(t *testing.T) {
	root := wz.NewDirectory("Mob", nil, []*wz.Image{
		wz.NewParsedImage("0100100", nil),     // shipping form (no .img)
		wz.NewParsedImage("0100101", nil),     // shipping form (no .img)
		wz.NewParsedImage("MobSkill", nil),    // sibling Mob.wz non-id image
		wz.NewParsedImage("0100102.img", nil), // legacy XML-walker form
	})
	file := wz.NewFileWithRoot("Mob", root)

	var got []uint32
	for _, img := range file.Root().Images() {
		if id, ok := imgID(img.Name()); ok {
			got = append(got, id)
		}
	}
	want := []uint32{100100, 100101, 100102}
	if len(got) != len(want) {
		t.Fatalf("imgID-parseable ids = %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("ids[%d] = %d, want %d", i, got[i], v)
		}
	}
}

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
