package summon

import (
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// FaithfulMaxPerHit is a faithful Go port of Cosmic's per-hit summon damage
// ceiling: SummonDamageHandler.calcMaxDamage (Cosmic
// src/main/java/net/server/channel/handlers/SummonDamageHandler.java:123-145)
// composed with Character.calculateMaxBaseDamage (Character.java:792-809) and
// Character.calculateMaxBaseMagicDamage (Character.java:832-850).
//
// All stats are the owner's session-effective (buffed) values, matching
// Cosmic's localstr/localdex/localluk and getTotalInt()/getTotalWatk()/
// getTotalMagic(). effWeaponAttack/effMagicAttack are the summon skill effect's
// watk/matk. weaponType is the owner's equipped main-weapon type
// (item.WeaponTypeNone ⇒ Cosmic's no-weapon fallback to SWORD1H).
//
// Parity note on the thief dagger case: Cosmic remaps DAGGER_OTHER→DAGGER_THIEVES
// (4.0→3.6) only when the owner is a thief. atlas-effective-stats does not carry
// job, so we use the DAGGER_OTHER multiplier (4.0) unconditionally. This only
// raises the ceiling for thief daggers, never lowers it — the clamp stays
// clamp-and-continue and never zeroes legitimate damage. Documented, intentional.
func FaithfulMaxPerHit(magic bool, totalWatk, totalMatk, totalInt, str, dex, luk uint32, weaponType item.WeaponType, effWeaponAttack, effMagicAttack int16) int64 {
	if magic {
		// Character.calculateMaxBaseMagicDamage(matk), matk = max(totalMagic, 14).
		matk := totalMatk
		if matk < 14 {
			matk = 14
		}
		base := calculateMaxBaseMagicDamage(int(matk), int(totalInt))
		// calcMaxDamage:128 — maxDamage = base * (0.05 * effect.getMatk()); double math.
		maxDamage := float64(base) * (0.05 * float64(effMagicAttack))
		return int64(maxDamage)
	}

	// Character.calculateMaxBaseDamage(watk, weaponType), watk = max(totalWatk, 14).
	watk := totalWatk
	if watk < 14 {
		watk = 14
	}
	maxBaseDmg := calculateMaxBaseDamage(int(watk), int(str), int(dex), int(luk), weaponType)

	// calcMaxDamage:140-141 — float32 mod, then maxBaseDmg(int) * (mod * effWatk).
	var mod float32 = 0.077
	if maxBaseDmg >= 438 {
		mod = 0.054
	}
	// Java: float maxDamage = maxBaseDmg * (mod * effWatk); (float arithmetic),
	// widened to double, then (int) truncation. Reproduce in float32.
	maxDamage := float32(maxBaseDmg) * (mod * float32(effWeaponAttack))
	return int64(maxDamage)
}

// cosmicWeaponMultiplier maps an Atlas item.WeaponType to Cosmic's
// WeaponType.getMaxDamageMultiplier() (WeaponType.java:24-54), restricted to the
// values ItemInformationProvider.getWeaponType (Atlas item.GetWeaponType) can
// actually return — Cosmic's getWeaponType only ever yields the *_SWING variants
// for general/2H weapons, so Atlas' collapsed enum maps 1:1:
//
//	OneHandedSword → SWORD1H        4.0
//	OneHandedAxe   → GENERAL1H_SWING 4.4
//	OneHandedMace  → GENERAL1H_SWING 4.4
//	Dagger         → DAGGER_OTHER    4.0  (thief remap to 3.6 omitted; see FaithfulMaxPerHit)
//	Wand           → WAND            3.6
//	Staff          → STAFF           3.6
//	TwoHandedSword → SWORD2H         4.6
//	TwoHandedAxe   → GENERAL2H_SWING 4.8
//	TwoHandedMace  → GENERAL2H_SWING 4.8
//	Spear          → SPEAR_STAB      5.0
//	Polearm        → POLE_ARM_SWING  5.0
//	Bow            → BOW             3.4
//	Crossbow       → CROSSBOW        3.6
//	Claw           → CLAW            3.6
//	Knuckle        → KNUCKLE         4.8
//	Gun            → GUN             3.6
//	None           → SWORD1H         4.0  (Cosmic no-weapon fallback, calcMaxDamage:137)
func cosmicWeaponMultiplier(w item.WeaponType) float64 {
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
		// item.WeaponTypeNone / no weapon equipped → Cosmic SWORD1H fallback.
		return 4.0
	}
}

// calculateMaxBaseDamage ports Character.calculateMaxBaseDamage(watk, weapon)
// (Character.java:792-809). mainstat/secondarystat selection is weapon-type
// dependent; the result is ceil(((mult*main + secondary)/100.0) * watk).
func calculateMaxBaseDamage(watk, str, dex, luk int, weapon item.WeaponType) int {
	var mainstat, secondarystat int
	switch weapon {
	case item.WeaponTypeBow, item.WeaponTypeCrossbow, item.WeaponTypeGun:
		mainstat = dex
		secondarystat = str
	case item.WeaponTypeClaw:
		// Cosmic CLAW (and DAGGER_THIEVES) → luk main, dex+str secondary.
		mainstat = luk
		secondarystat = dex + str
	default:
		// SWORD/AXE/MACE/SPEAR/POLEARM/STAFF/WAND/KNUCKLE and DAGGER (non-thief,
		// DAGGER_OTHER) all use str main, dex secondary.
		mainstat = str
		secondarystat = dex
	}
	mult := cosmicWeaponMultiplier(weapon)
	return int(math.Ceil(((mult*float64(mainstat) + float64(secondarystat)) / 100.0) * float64(watk)))
}

// calculateMaxBaseMagicDamage ports Character.calculateMaxBaseMagicDamage(matk)
// (Character.java:832-850). totalInt is the owner's effective INT.
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
