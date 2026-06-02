package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
)

// TestParseSlotCodesPortedFromDonor pins the two-character code splitter to
// the same shape the donor's parseSlotCodes test asserted. Donor:
// characterimage/vslot_test.go:5-28.
func TestParseSlotCodesPortedFromDonor(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"X", nil},
		{"Cp", []string{"Cp"}},
		{"CpH5", []string{"Cp", "H5"}},
		{"CpHdH1H2", []string{"Cp", "Hd", "H1", "H2"}},
	}
	for _, tc := range tests {
		got := parseSlotCodes(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("parseSlotCodes(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i, c := range got {
			if c != tc.want[i] {
				t.Errorf("parseSlotCodes(%q)[%d] = %q, want %q", tc.in, i, c, tc.want[i])
			}
		}
	}
}

// TestClaimSlotsHigherPriorityWins exercises the precedence chain: the cap
// (ownerEquipment) is processed before the hair (ownerHair) and so owns
// every contested slot. Slots that only the hair claims fall to the hair.
// Donor: characterimage/vslot_test.go:30-47.
func TestClaimSlotsHigherPriorityWins(t *testing.T) {
	owners := []vslotOwner{
		{templateID: 1002357, vslot: "CpH1H2H3H4H5HfHsHbAe", kind: ownerEquipment},
		{templateID: 30030, vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
	}
	claimed := claimSlots(owners)

	for _, code := range []string{"Cp", "H1", "H2", "H3", "H4", "H5", "Hf", "Hs", "Hb", "Ae"} {
		if got := claimed[code]; got != 1002357 {
			t.Errorf("slot %q: want owner 1002357, got %d", code, got)
		}
	}
	if got := claimed["H6"]; got != 30030 {
		t.Errorf("slot H6: want owner 30030, got %d", got)
	}
}

// TestApplyVslotOcclusionFullHelmet locks in the exact case the donor
// flagged: a full helmet's claim of every hair slot (H1..H5, Hf, Hs, Hb)
// causes every hair part whose layer is mapped to one of those codes to be
// suppressed. The cap and body remain. Donor:
// characterimage/vslot_test.go:49-80.
func TestApplyVslotOcclusionFullHelmet(t *testing.T) {
	smap := map[string]string{
		"cap":           "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"hair":          "H2",
		"hairOverHead":  "H1",
		"hairBelowBody": "Hb",
		"body":          "Bd",
	}
	placed := []placement{
		{templateID: 2000, sprite: manifest.Sprite{Part: "body", Z: "body"}},
		{templateID: 1002357, sprite: manifest.Sprite{Part: "cap", Z: "cap"}},
		{templateID: 30030, sprite: manifest.Sprite{Part: "hair", Z: "hair"}},
		{templateID: 30030, sprite: manifest.Sprite{Part: "hairOverHead", Z: "hairOverHead"}},
		{templateID: 30030, sprite: manifest.Sprite{Part: "hairBelowBody", Z: "hairBelowBody"}},
	}
	owners := []vslotOwner{
		{templateID: 1002357, vslot: "CpH1H2H3H4H5HfHsHbAe", kind: ownerEquipment},
		{templateID: 30030, vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
		{templateID: 2000, vslot: "Bd", kind: ownerBody},
	}
	out := applyVslotOcclusion(placed, smap, owners)

	want := map[string]bool{"body": true, "cap": true}
	if len(out) != len(want) {
		t.Fatalf("kept %d parts, want %d: %+v", len(out), len(want), out)
	}
	for _, p := range out {
		if !want[p.sprite.Part] {
			t.Errorf("unexpected part kept: part=%q owner=%d", p.sprite.Part, p.templateID)
		}
	}
}

// TestApplyVslotOcclusionBasicCap covers the half-helmet case: a cap that
// claims only "Cp" and "H5" leaves the bangs (H1..H4) intact. All four hair
// parts in the placement list must survive. Donor:
// characterimage/vslot_test.go:82-104.
func TestApplyVslotOcclusionBasicCap(t *testing.T) {
	smap := map[string]string{
		"cap":           "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"hair":          "H2",
		"hairOverHead":  "H1",
		"hairBelowBody": "Hb",
	}
	placed := []placement{
		{templateID: 1002069, sprite: manifest.Sprite{Part: "cap", Z: "cap"}},
		{templateID: 30020, sprite: manifest.Sprite{Part: "hair", Z: "hair"}},
		{templateID: 30020, sprite: manifest.Sprite{Part: "hairOverHead", Z: "hairOverHead"}},
		{templateID: 30020, sprite: manifest.Sprite{Part: "hairBelowBody", Z: "hairBelowBody"}},
	}
	owners := []vslotOwner{
		{templateID: 1002069, vslot: "CpH5", kind: ownerEquipment},
		{templateID: 30020, vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
	}
	out := applyVslotOcclusion(placed, smap, owners)

	if len(out) != 4 {
		t.Fatalf("kept %d parts, want 4: %+v", len(out), out)
	}
}

// TestApplyVslotOcclusionUnknownLayerKept locks in the safety property: a
// part whose layer name isn't in the smap is never dropped — we'd rather
// over-render than blank a body part to a stale smap. Donor:
// characterimage/vslot_test.go:106-119.
func TestApplyVslotOcclusionUnknownLayerKept(t *testing.T) {
	smap := map[string]string{"body": "Bd"}
	placed := []placement{
		{templateID: 2000, sprite: manifest.Sprite{Part: "body", Z: "body"}},
		{templateID: 2000, sprite: manifest.Sprite{Part: "arm", Z: "arm"}}, // arm not in smap
	}
	owners := []vslotOwner{
		{templateID: 2000, vslot: "Bd", kind: ownerBody},
	}
	out := applyVslotOcclusion(placed, smap, owners)
	if len(out) != 2 {
		t.Fatalf("kept %d parts, want 2 (unknown smap layers default visible)", len(out))
	}
}

// TestApplyVslotOcclusionEmptySmapNoOp confirms the early-out: a fetch
// failure that returns an empty smap (no entries) must NOT silently drop
// every placement. Atlas-renders depends on this to keep rendering when
// the smap sidecar is unavailable.
func TestApplyVslotOcclusionEmptySmapNoOp(t *testing.T) {
	placed := []placement{
		{templateID: 2000, sprite: manifest.Sprite{Part: "body", Z: "body"}},
		{templateID: 30030, sprite: manifest.Sprite{Part: "hair", Z: "hair"}},
	}
	owners := []vslotOwner{
		{templateID: 1002357, vslot: "CpH1", kind: ownerEquipment},
	}
	out := applyVslotOcclusion(placed, nil, owners)
	if len(out) != 2 {
		t.Fatalf("empty smap: kept %d, want 2 (no-op fallback)", len(out))
	}
}

// TestAppendOwnerSkipsEmptyVslot pins the donor's contract: a template
// with no vslot string contributes no owner entry, so its templates can
// still be placed without spuriously occluding others (and itself isn't
// dropped because applyVslotOcclusion needs at least one claim for the
// part's own templateID match to fire).
func TestAppendOwnerSkipsEmptyVslot(t *testing.T) {
	owners := []vslotOwner{}
	owners = appendOwner(owners, 12345, "", ownerEquipment)
	if len(owners) != 0 {
		t.Errorf("empty vslot should be skipped, got %+v", owners)
	}
	owners = appendOwner(owners, 12345, "CpH5", ownerEquipment)
	if len(owners) != 1 || owners[0].vslot != "CpH5" {
		t.Errorf("non-empty vslot should be appended, got %+v", owners)
	}
}

// TestSmapKeyCaseInsensitiveFallback exercises the case-insensitive lookup
// path used when a part's layer name doesn't match the smap's canonical
// (lowercase) form exactly. Donor: characterimage/vslot.go:126-136.
func TestSmapKeyCaseInsensitiveFallback(t *testing.T) {
	smap := map[string]string{"hairoverhead": "H1"}
	if got := smapKey(smap, "hairOverHead"); got != "H1" {
		t.Errorf("smapKey case-fold = %q, want H1", got)
	}
	if got := smapKey(smap, "missing"); got != "" {
		t.Errorf("smapKey miss = %q, want empty", got)
	}
}
