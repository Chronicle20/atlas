package summon

// ConservativeMaxPerHit is an interim per-hit ceiling pending the full weapon-type
// port (see Task 3.6 / design.md §8.3). It bounds damage by the attack-multiplier
// term of Cosmic's formula using a generous base-damage proxy, so blatant client
// inflation is clamped while legitimate hits pass. The exact Cosmic formula is the
// parity target and replaces this in Task 3.6.
func ConservativeMaxPerHit(magic bool, totalWatk, totalMatk uint32, effWeaponAttack, effMagicAttack int16) int64 {
	if magic {
		matk := totalMatk
		if matk < 14 {
			matk = 14
		}
		// generous proxy for maxBaseMagicDamage: matk * matk (Cosmic squares matk-ish term)
		base := int64(matk) * int64(matk)
		return base * 5 / 100 * int64(effMagicAttack)
	}
	watk := totalWatk
	if watk < 14 {
		watk = 14
	}
	// generous proxy for maxBaseDamage: a high multiplier on watk (4x covers high-mastery)
	base := int64(watk) * 4
	mod := int64(77) // 0.077 * 1000
	if base >= 438 {
		mod = 54 // 0.054 * 1000
	}
	return base * mod / 1000 * int64(effWeaponAttack)
}

func clampDamage(reported uint32, max int64) uint32 {
	if max <= 0 {
		return reported // no ceiling computable (e.g. stats fetch failed); do not clamp to 0
	}
	if int64(reported) > max {
		return uint32(max)
	}
	return reported
}
