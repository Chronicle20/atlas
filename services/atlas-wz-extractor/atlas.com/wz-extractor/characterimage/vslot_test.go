package characterimage

import "testing"

func TestParseSlotCodes(t *testing.T) {
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

func TestClaimSlotsHigherPriorityWins(t *testing.T) {
	owners := []vslotOwner{
		{templateId: "1002357", vslot: "CpH1H2H3H4H5HfHsHbAe", kind: ownerEquipment},
		{templateId: "30030", vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
	}
	claimed := claimSlots(owners)

	// Cap was processed first → owns every contested slot.
	for _, code := range []string{"Cp", "H1", "H2", "H3", "H4", "H5", "Hf", "Hs", "Hb", "Ae"} {
		if got := claimed[code]; got != "1002357" {
			t.Errorf("slot %q: want owner 1002357, got %q", code, got)
		}
	}
	// H6 was only claimed by hair → hair owns it.
	if got := claimed["H6"]; got != "30030" {
		t.Errorf("slot H6: want owner 30030, got %q", got)
	}
}

func TestApplyVslotOcclusionFullHelmet(t *testing.T) {
	smap := map[string]string{
		"cap":          "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"hair":         "H2",
		"hairOverHead": "H1",
		"hairBelowBody": "Hb",
		"body":         "Bd",
	}
	placed := []placement{
		{templateId: "2000", meta: PartMeta{Z: "body"}},
		{templateId: "1002357", meta: PartMeta{Z: "cap"}},
		{templateId: "30030", meta: PartMeta{Z: "hair"}},
		{templateId: "30030", meta: PartMeta{Z: "hairOverHead"}},
		{templateId: "30030", meta: PartMeta{Z: "hairBelowBody"}},
	}
	owners := []vslotOwner{
		{templateId: "1002357", vslot: "CpH1H2H3H4H5HfHsHbAe", kind: ownerEquipment},
		{templateId: "30030", vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
		{templateId: "2000", vslot: "Bd", kind: ownerBody},
	}
	out := applyVslotOcclusion(placed, smap, owners)

	want := map[string]bool{"body": true, "cap": true}
	if len(out) != len(want) {
		t.Fatalf("kept %d parts, want %d: %+v", len(out), len(want), out)
	}
	for _, p := range out {
		if !want[p.meta.Z] {
			t.Errorf("unexpected part kept: z=%q owner=%s", p.meta.Z, p.templateId)
		}
	}
}

func TestApplyVslotOcclusionBasicCap(t *testing.T) {
	smap := map[string]string{
		"cap":           "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"hair":          "H2",
		"hairOverHead":  "H1",
		"hairBelowBody": "Hb",
	}
	placed := []placement{
		{templateId: "1002069", meta: PartMeta{Z: "cap"}},
		{templateId: "30020", meta: PartMeta{Z: "hair"}},
		{templateId: "30020", meta: PartMeta{Z: "hairOverHead"}},
		{templateId: "30020", meta: PartMeta{Z: "hairBelowBody"}},
	}
	owners := []vslotOwner{
		{templateId: "1002069", vslot: "CpH5", kind: ownerEquipment},
		{templateId: "30020", vslot: "H1H2H3H4H5H6HfHsHb", kind: ownerHair},
	}
	out := applyVslotOcclusion(placed, smap, owners)

	if len(out) != 4 {
		t.Fatalf("kept %d parts, want 4: %+v", len(out), out)
	}
}

func TestApplyVslotOcclusionUnknownLayerKept(t *testing.T) {
	smap := map[string]string{"body": "Bd"}
	placed := []placement{
		{templateId: "2000", meta: PartMeta{Z: "body"}},
		{templateId: "2000", meta: PartMeta{Z: "arm"}}, // arm not in smap
	}
	owners := []vslotOwner{
		{templateId: "2000", vslot: "Bd", kind: ownerBody},
	}
	out := applyVslotOcclusion(placed, smap, owners)
	if len(out) != 2 {
		t.Fatalf("kept %d parts, want 2 (unknown smap layers default visible)", len(out))
	}
}
