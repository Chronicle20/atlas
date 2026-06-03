package character

// vslot/smap-based occlusion. Each equipment item's atlas manifest declares
// a `vslot` (e.g. a full helmet claims "CpH1H2H3H4H5HfHsHbAe"). Each layer
// name in the smap maps the layer's z-string to the slot codes it occupies
// (e.g. "hairOverHead" → "H1"). When an equipment item claims a slot that a
// hair or face part also occupies, the higher-priority item wins and the
// lower-priority part is suppressed. Without this filter, full helmets have
// their front bangs painted on top of the helmet.
//
// Priority order, highest-first: equipment → hair → face → head → body.
// Within equipment, ties are broken by insertion order (which matches the
// donor's iteration order over the filtered slot map).
//
// Donor:
//   - services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/
//     vslot.go (the algorithm)
//   - services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/
//     compositor.go:232-235 (the wiring into the composite pipeline)

import "strings"

// ownerKind ranks owner classes for slot-claim precedence — lower wins. The
// numeric ordering matches the donor's iota declaration verbatim. Donor:
// characterimage/vslot.go:21-28.
type ownerKind int

const (
	ownerEquipment ownerKind = iota
	ownerHair
	ownerFace
	ownerHead
	ownerBody
)

// vslotOwner pairs a templateId with the vslot string it claims and the
// ownership class used for precedence. Donor: characterimage/vslot.go:30-36.
type vslotOwner struct {
	templateID uint32
	vslot      string
	kind       ownerKind
}

// appendOwner records a templateId's vslot for occlusion resolution. A
// missing or empty vslot (typical of synthetic test fixtures or extension
// templates whose info block lacks vslot data) drops the owner silently —
// the part is still placed; it just contributes no slot claims.
// Donor: characterimage/vslot.go:42-48.
func appendOwner(owners []vslotOwner, templateID uint32, vslot string, kind ownerKind) []vslotOwner {
	if vslot == "" {
		return owners
	}
	return append(owners, vslotOwner{templateID: templateID, vslot: vslot, kind: kind})
}

// parseSlotCodes splits a vslot/smap value into two-character slot codes.
// Codes are two characters each by MapleStory convention (e.g. "Cp", "H1").
// An empty input or anything shorter than two characters returns nil.
// Donor: characterimage/vslot.go:53-62.
func parseSlotCodes(s string) []string {
	if len(s) < 2 {
		return nil
	}
	out := make([]string, 0, len(s)/2)
	for i := 0; i+2 <= len(s); i += 2 {
		out = append(out, s[i:i+2])
	}
	return out
}

// claimSlots walks the owners in precedence order, recording the first
// owner to claim each slot code. Later owners cannot displace earlier
// claims — that's how the cap "wins" hair slots when both list them.
// Donor: characterimage/vslot.go:67-77.
func claimSlots(owners []vslotOwner) map[string]uint32 {
	claimed := make(map[string]uint32)
	for _, o := range owners {
		for _, code := range parseSlotCodes(o.vslot) {
			if _, exists := claimed[code]; !exists {
				claimed[code] = o.templateID
			}
		}
	}
	return claimed
}

// applyVslotOcclusion drops any placement whose smap-resolved slots are
// all owned by some other template. A part survives if at least one of the
// slot codes its layer occupies is owned by its own template, or if the
// layer is unknown to the smap (which keeps body/extension parts rendering
// when no slot info is available).
//
// The owners slice must already be sorted by precedence (caller sorts by
// ownerKind ascending — equipment first). Donor:
// characterimage/vslot.go:84-99.
func applyVslotOcclusion(placed []placement, smap map[string]string, owners []vslotOwner) []placement {
	if len(placed) == 0 || len(smap) == 0 {
		return placed
	}
	claimed := claimSlots(owners)
	if len(claimed) == 0 {
		return placed
	}
	out := placed[:0]
	for _, p := range placed {
		if isPartVisible(p, claimed, smap) {
			out = append(out, p)
		}
	}
	return out
}

// isPartVisible returns true if the part should be drawn under the claim
// map. Parts whose layer is missing from the smap (or whose smap value is
// empty) default to visible — we never want to drop body, head, or
// fixture-synthesized parts on a missing slot lookup.
//
// The smap is keyed on the render-layer label (the WZ canvas `z` child),
// carried by manifest.Sprite.Z — NOT by Part (the canvas name). The donor
// looks up smap by meta.Z (characterimage/vslot.go:105-120,
// characterimage/compositor.go isPartVisible(p.meta.Z)); keying on Part
// mislabels any part whose canvas name differs from its z-label.
func isPartVisible(p placement, claimed map[string]uint32, smap map[string]string) bool {
	z := smapKey(smap, string(p.sprite.Z))
	if z == "" {
		return true
	}
	codes := parseSlotCodes(z)
	if len(codes) == 0 {
		return true
	}
	for _, code := range codes {
		if claimed[code] == p.templateID {
			return true
		}
	}
	return false
}

// smapKey looks up the slot string for a layer name, falling through a
// case-insensitive match if the exact key is missing. The extracted smap
// is lowercase in shipped data; synthetic test fixtures occasionally use
// mixed case (e.g. "hairOverHead"). Donor: characterimage/vslot.go:126-136.
func smapKey(smap map[string]string, z string) string {
	if v, ok := smap[z]; ok {
		return v
	}
	for k, v := range smap {
		if strings.EqualFold(k, z) {
			return v
		}
	}
	return ""
}
