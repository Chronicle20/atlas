package handler

import (
	"sort"

	"atlas-channel/asset"
	"atlas-channel/data/skill/effect/statup"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// StarDraw is one slot-level consume of a chosen throwing star for the
// Shadow Stars cast cost.
type StarDraw struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

// validateShadowStar reports whether starItemId is a throwing-star
// classification AND present (quantity > 0) in the caster's consumable assets.
func validateShadowStar(assets []asset.Model, starItemId uint32) bool {
	if !item.IsThrowingStar(item.Id(starItemId)) {
		return false
	}
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			return true
		}
	}
	return false
}

// resolveStarConsume draws `count` of exactly starItemId across ascending
// consumable slots. `available` is the sum of planned draws
// (min(count, total owned)); available < count signals a shortfall.
func resolveStarConsume(assets []asset.Model, starItemId uint32, count int) (draws []StarDraw, available int) {
	matching := make([]asset.Model, 0, len(assets))
	for _, a := range assets {
		if a.TemplateId() == starItemId && a.Quantity() > 0 {
			matching = append(matching, a)
		}
	}
	if len(matching) == 0 || count <= 0 {
		return nil, 0
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })

	remaining := count
	draws = make([]StarDraw, 0, len(matching))
	for _, a := range matching {
		if remaining <= 0 {
			break
		}
		draw := int(a.Quantity())
		if draw > remaining {
			draw = remaining
		}
		draws = append(draws, StarDraw{Slot: a.Slot(), ItemId: starItemId, Quantity: int16(draw)})
		remaining -= draw
		available += draw
	}
	return draws, available
}

// rewriteShadowClawStatups returns a copy of statups with the SHADOW_CLAW
// entry's amount set to starItemId, preserving any other statups. atlas-data's
// produceBuffStatAmount drops zero-value statups (its `if value != 0` guard),
// so the SHADOW_CLAW placeholder never survives the reader for 4121006 — the
// statups atlas-channel actually receives carry NO SHADOW_CLAW entry. If the
// entry is absent, one is appended so the star id always reaches the buff.
// Mirrors mount.go's tamedMountStatups for MONSTER_RIDING.
func rewriteShadowClawStatups(statups []statup.Model, starItemId uint32) []statup.Model {
	out := make([]statup.Model, 0, len(statups)+1)
	hasClaw := false
	for _, su := range statups {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			out = append(out, statup.NewModel(su.Mask(), int32(starItemId)))
			hasClaw = true
			continue
		}
		out = append(out, su)
	}
	if !hasClaw {
		out = append(out, statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), int32(starItemId)))
	}
	return out
}

// resolveShadowStarsCast validates the chosen star and resolves the buff
// statups + consume draws for a Shadow Stars cast. ok=false means the star is
// invalid (wrong classification or not owned) and the cast MUST abort — the
// returned rewritten/draws are nil. shortfall reports available < bulletCount.
func resolveShadowStarsCast(assets []asset.Model, statups []statup.Model, starItemId uint32, bulletCount int) (rewritten []statup.Model, draws []StarDraw, shortfall bool, ok bool) {
	if !validateShadowStar(assets, starItemId) {
		return nil, nil, false, false
	}
	draws, available := resolveStarConsume(assets, starItemId, bulletCount)
	rewritten = rewriteShadowClawStatups(statups, starItemId)
	return rewritten, draws, available < bulletCount, true
}
