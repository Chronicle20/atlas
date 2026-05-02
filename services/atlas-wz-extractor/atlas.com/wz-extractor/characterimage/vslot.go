package characterimage

// vslot/smap-based occlusion. Each item's info.json declares a `vslot` —
// a string of two-character slot codes the item claims (e.g. a full helmet
// claims "CpH1H2H3H4H5HfHsHbAe"). Each layer name has a smap entry that
// maps the layer's z-string to the slot codes it occupies (e.g.
// hairOverHead -> "H1"). When an equipment item claims a slot that a hair
// or face part also occupies, the higher-priority item wins and the
// lower-priority part is suppressed. Without this filter, full helmets
// have their front bangs painted on top of the helmet.
//
// Priority order, highest-first: equipment → hair → face → head → body.
// Within equipment, ties are broken by template id ordering, which is
// stable enough for the slot conflicts that actually occur in shipped
// data (cap vs. coat vs. weapon claim disjoint slot codes).

import "strings"

// ownerKind ranks owner classes for slot-claim precedence — lower wins.
type ownerKind int

const (
	ownerEquipment ownerKind = iota
	ownerHair
	ownerFace
	ownerHead
	ownerBody
)

// vslotOwner pairs a templateId with the vslot string it claims and the
// ownership class used for precedence.
type vslotOwner struct {
	templateId string
	vslot      string
	kind       ownerKind
}

// appendOwner records a templateId's vslot for occlusion resolution.
// A missing info.json (typical of synthetic test fixtures or extension
// templates) drops the owner silently — the part is still placed; it
// just contributes no slot claims.
func appendOwner(owners []vslotOwner, c *Compositor, assetsRoot, templateId string, kind ownerKind) []vslotOwner {
	info, err := c.cache.info(assetsRoot, templateId)
	if err != nil || info.Vslot == "" {
		return owners
	}
	return append(owners, vslotOwner{templateId: templateId, vslot: info.Vslot, kind: kind})
}

// parseSlotCodes splits a vslot/smap value into two-character slot codes.
// Codes are two characters each by MapleStory convention (e.g. "Cp", "H1").
// An empty input returns nil.
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
func claimSlots(owners []vslotOwner) map[string]string {
	claimed := make(map[string]string)
	for _, o := range owners {
		for _, code := range parseSlotCodes(o.vslot) {
			if _, exists := claimed[code]; !exists {
				claimed[code] = o.templateId
			}
		}
	}
	return claimed
}

// applyVslotOcclusion drops any placement whose smap-resolved slots are
// all owned by some other template. A part survives if at least one of
// the slot codes its layer occupies is owned by its own template, or
// if the layer is unknown to the smap (which keeps body/extension parts
// rendering when no slot info is available).
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

// isPartVisible returns true if the part should be drawn under the
// claim map. Parts whose layer is missing from the smap (or whose smap
// value is empty) default to visible — we never want to drop body, head,
// or fixture-synthesized parts on a missing slot lookup.
func isPartVisible(p placement, claimed map[string]string, smap map[string]string) bool {
	z := smapKey(smap, p.meta.Z)
	if z == "" {
		return true
	}
	codes := parseSlotCodes(z)
	if len(codes) == 0 {
		return true
	}
	for _, code := range codes {
		if claimed[code] == p.templateId {
			return true
		}
	}
	return false
}

// smapKey looks up the slot string for a z-name, falling through a
// case-insensitive match if the exact key is missing. The extracted
// smap is lowercase; z values from extracted part metadata are also
// lowercase, but synthetic test fixtures occasionally use mixed case.
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
