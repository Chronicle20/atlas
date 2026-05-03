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

// HealXp computes the experience awarded to the caster from the heal
// portion of the cast. Per recipient, the contribution is
// min(perTarget, MaxHp - Hp); the sum is divided by 10 and multiplied
// by the skill level. Returns 0 on any pathological negative result.
func HealXp(perTarget int16, recipients []recipient, skillLevel byte) uint32 {
	var total int64
	for _, r := range recipients {
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
		total += int64(applied)
	}
	xp := total / 10 * int64(skillLevel)
	if xp < 0 {
		return 0
	}
	return uint32(xp)
}
