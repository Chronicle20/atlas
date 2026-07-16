package summon

import (
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// FaithfulMaxPerHit computes the v83-era per-hit summon damage ceiling: the
// owner's max base damage (weapon-type-aware for physical, INT-curve-based for
// magic) scaled by the summon skill effect's watk/matk.
//
// All stats are the owner's session-effective (buffed) values — buffed
// str/dex/luk plus total INT/watk/matk including equips and buffs.
// effWeaponAttack/effMagicAttack are the summon skill effect's
// watk/matk. weaponType is the owner's equipped main-weapon type
// (item.WeaponTypeNone ⇒ the no-weapon fallback to the one-handed-sword
// multiplier).
//
// Note on the thief dagger case: a dagger in a thief's hands uses the
// luk-main 3.6 multiplier rather than the str-main 4.0 one, but
// atlas-effective-stats does not carry job, so we use the non-thief dagger
// multiplier (4.0) unconditionally. This only raises the ceiling for thief
// daggers, never lowers it — the clamp stays clamp-and-continue and never
// zeroes legitimate damage. Documented, intentional.
func FaithfulMaxPerHit(magic bool, totalWatk, totalMatk, totalInt, str, dex, luk uint32, weaponType item.WeaponType, effWeaponAttack, effMagicAttack int16) int64 {
	if magic {
		// Magic branch: matk floor of 14, then the INT-curve base damage.
		matk := totalMatk
		if matk < 14 {
			matk = 14
		}
		base := calculateMaxBaseMagicDamage(int(matk), int(totalInt))
		// maxDamage = base * (0.05 * effect matk); double math.
		maxDamage := float64(base) * (0.05 * float64(effMagicAttack))
		return int64(maxDamage)
	}

	// Physical branch: watk floor of 14, then the weapon-type-aware base damage.
	watk := totalWatk
	if watk < 14 {
		watk = 14
	}
	maxBaseDmg := calculateMaxBaseDamage(int(watk), int(str), int(dex), int(luk), weaponType)

	// Summon damage modifier: 0.077 below a 438 base-damage threshold, 0.054
	// at or above it. The multiply is done in float32 (mod * effWatk first),
	// then truncated to int — reproduce that exact arithmetic.
	var mod float32 = 0.077
	if maxBaseDmg >= 438 {
		mod = 0.054
	}
	maxDamage := float32(maxBaseDmg) * (mod * float32(effWeaponAttack))
	return int64(maxDamage)
}

// weaponTypeMultiplier maps an Atlas item.WeaponType to the v83-era
// max-damage multiplier for that weapon class. Axes/maces/polearms use their
// swing-variant multipliers (the stab variants never apply to max-damage
// computation for these classes), so Atlas' collapsed enum maps 1:1:
//
//	OneHandedSword → 4.0
//	OneHandedAxe   → 4.4
//	OneHandedMace  → 4.4
//	Dagger         → 4.0  (thief remap to 3.6 omitted; see FaithfulMaxPerHit)
//	Wand           → 3.6
//	Staff          → 3.6
//	TwoHandedSword → 4.6
//	TwoHandedAxe   → 4.8
//	TwoHandedMace  → 4.8
//	Spear          → 5.0
//	Polearm        → 5.0
//	Bow            → 3.4
//	Crossbow       → 3.6
//	Claw           → 3.6
//	Knuckle        → 4.8
//	Gun            → 3.6
//	None           → 4.0  (no-weapon fallback = one-handed sword)
func weaponTypeMultiplier(w item.WeaponType) float64 {
	switch w {
	case item.WeaponTypeOneHandedSword:
		return 4.0
	case item.WeaponTypeOneHandedAxe, item.WeaponTypeOneHandedMace:
		return 4.4
	case item.WeaponTypeDagger:
		return 4.0
	case item.WeaponTypeWand, item.WeaponTypeStaff:
		return 3.6
	case item.WeaponTypeTwoHandedSword:
		return 4.6
	case item.WeaponTypeTwoHandedAxe, item.WeaponTypeTwoHandedMace:
		return 4.8
	case item.WeaponTypeSpear:
		return 5.0
	case item.WeaponTypePolearm:
		return 5.0
	case item.WeaponTypeBow:
		return 3.4
	case item.WeaponTypeCrossbow:
		return 3.6
	case item.WeaponTypeClaw:
		return 3.6
	case item.WeaponTypeKnuckle:
		return 4.8
	case item.WeaponTypeGun:
		return 3.6
	default:
		// item.WeaponTypeNone / no weapon equipped → one-handed-sword fallback.
		return 4.0
	}
}

// calculateMaxBaseDamage computes the owner's max base physical damage.
// mainstat/secondarystat selection is weapon-type
// dependent; the result is ceil(((mult*main + secondary)/100.0) * watk).
func calculateMaxBaseDamage(watk, str, dex, luk int, weapon item.WeaponType) int {
	var mainstat, secondarystat int
	switch weapon {
	case item.WeaponTypeBow, item.WeaponTypeCrossbow, item.WeaponTypeGun:
		mainstat = dex
		secondarystat = str
	case item.WeaponTypeClaw:
		// Claw (and a dagger in thief hands) → luk main, dex+str secondary.
		mainstat = luk
		secondarystat = dex + str
	default:
		// SWORD/AXE/MACE/SPEAR/POLEARM/STAFF/WAND/KNUCKLE and DAGGER (non-thief,
		// DAGGER_OTHER) all use str main, dex secondary.
		mainstat = str
		secondarystat = dex
	}
	mult := weaponTypeMultiplier(weapon)
	return int(math.Ceil(((mult*float64(mainstat) + float64(secondarystat)) / 100.0) * float64(watk)))
}

// calculateMaxBaseMagicDamage computes the owner's max base magic damage from
// matk and INT via the piecewise INT curve (breakpoints at INT 1700 and 2000),
// then scales by 107/100. totalInt is the owner's effective INT.
func calculateMaxBaseMagicDamage(matk, totalInt int) int {
	maxbasedamage := matk
	if totalInt > 2000 {
		maxbasedamage -= 2000
		maxbasedamage += int((0.09033024267 * float64(totalInt)) + 3823.8038)
	} else {
		maxbasedamage -= totalInt
		if totalInt > 1700 {
			maxbasedamage += int(0.1996049769 * math.Pow(float64(totalInt), 1.300631341))
		} else {
			maxbasedamage += int(0.1996049769 * math.Pow(float64(totalInt), 1.290631341))
		}
	}
	return (maxbasedamage * 107) / 100
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
