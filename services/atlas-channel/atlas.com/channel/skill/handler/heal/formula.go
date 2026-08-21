package heal

import "math"

// recipient is the per-target snapshot used by both the formula
// (HealXp's missing-HP cap) and the apply step. Position is captured
// for tests that exercise recipient selection independent of XP math.
type recipient struct {
	Id       uint32
	X        int16
	Y        int16
	Hp       uint16
	MaxHp    uint16
	IsCaster bool
}

// HealAmount returns the per-target HP delta (clamped to int16) for a
// Heal cast. Variance is injected so tests can pin the result.
//
//	base = skillHpPct * (MA*1.5 + INT*0.8) / 100
//	perTarget = floor(base * variance / partyTargets)
//
// partyTargets is clamped to a minimum of 1; negative perTarget clamps
// to 0; overflow above int16 max clamps to int16 max.
func HealAmount(skillHpPct uint16, magicAttack, intelligence, partyTargets int, variance float64) int16 {
	if partyTargets < 1 {
		partyTargets = 1
	}
	base := float64(skillHpPct) * (float64(magicAttack)*1.5 + float64(intelligence)*0.8) / 100.0
	perTarget := math.Floor(base * variance / float64(partyTargets))
	if perTarget < 0 {
		return 0
	}
	if perTarget > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(perTarget)
}

// appliedPerRecipient returns the actual HP delta a Heal cast lands
// on one recipient: never more than what's missing, never below zero.
// Shared between the apply path (so we don't push HP past MaxHp and
// risk an overflow / DIED in atlas-character's enforceBounds) and the
// HealXp accumulator (so XP credit matches the HP that actually
// landed).
func appliedPerRecipient(perTarget int16, r recipient) int16 {
	missing := int32(r.MaxHp) - int32(r.Hp)
	if missing < 0 {
		missing = 0
	}
	applied := int32(perTarget)
	if applied > missing {
		applied = missing
	}
	if applied < 0 {
		applied = 0
	}
	return int16(applied)
}

// healDelta returns the ChangeHP delta for one recipient of a Heal cast.
//
// Non-zombified: the existing headroom clamp -- never push Hp past MaxHp.
// Zombified: the reference negates the heal (StatEffect.calcHPChange), so the
// delta is damage. It is clamped to the recipient's CURRENT Hp so a cast never
// removes more HP than the recipient has; landing exactly on 0 kills them,
// which is intended (atlas-character emits DIED at adjusted == 0).
// appliedPerRecipient is deliberately never handed a negative value: its
// headroom clamp would mangle one. (task-256 FR-12/FR-13/FR-14)
func healDelta(perTarget int16, r recipient, zombified bool) int16 {
	if !zombified {
		return appliedPerRecipient(perTarget, r)
	}
	if r.Hp == 0 {
		return 0
	}
	magnitude := int32(perTarget)
	if magnitude < 0 {
		magnitude = 0
	}
	if magnitude > int32(r.Hp) {
		magnitude = int32(r.Hp)
	}
	// Unreachable from today's inputs: perTarget is int16, so
	// -magnitude >= -32767 > math.MinInt16. Kept as a defensive guard
	// against a future widening of perTarget's type.
	if -magnitude < math.MinInt16 {
		return math.MinInt16
	}
	return int16(-magnitude)
}

// HealXp computes the experience awarded to the caster from the heal
// portion of the cast. Per recipient, the contribution is the amount
// actually applied (see appliedPerRecipient); the sum is divided by 10
// and multiplied by the skill level. Returns 0 on any pathological
// negative result.
func HealXp(perTarget int16, recipients []recipient, skillLevel byte) uint32 {
	var total int64
	for _, r := range recipients {
		total += int64(appliedPerRecipient(perTarget, r))
	}
	xp := total / 10 * int64(skillLevel)
	if xp < 0 {
		return 0
	}
	return uint32(xp)
}
