package handler

import (
	"math"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	// maxLegitimateDamage is the client's own hard damage cap in every
	// verified version's CalcDamage (design task-157 §8).
	maxLegitimateDamage = int32(999999)

	// reflect attack types feed atlas-monsters' Damage command; the values
	// mirror packetmodel.AttackTypeMelee / AttackTypeMagic, which is what
	// the mob-side counter-buff check distinguishes on.
	reflectAttackTypePhysical = byte(packetmodel.AttackTypeMelee)
	reflectAttackTypeMagic    = byte(packetmodel.AttackTypeMagic)
)

// mitigationInput carries everything computeMitigation needs, with wire
// cross-checks already validated against server-side buff state and all
// tenant-version gates pre-resolved, so the math is a pure function.
type mitigationInput struct {
	attackIdx  packetmodel.DamageType
	rawDamage  int32 // already clamped, >= 0
	mobSourced bool  // attackIdx >= DamageTypePhysical

	// powerGuardSignal: wire isPowerGuard AND server-side POWER_GUARD buff
	// AND mob-sourced. manaReflectSignal: wire reflect echo without
	// isPowerGuard AND server-side MANA_REFLECTION buff AND mob skill
	// attack (attackIdx >= 0). The client rolls Mana Reflection's prop;
	// the validated signal is honored, amounts are always recomputed.
	powerGuardSignal  bool
	manaReflectSignal bool

	currentMP uint16
	meso      uint32

	// Buff statup amounts (0 = buff absent).
	magicGuardPct        int32
	infinity             bool
	powerGuardPct        int32
	mesoGuardPct         int32
	comboBarrierPermille int32
	magicShieldPct       int32

	// Passive/effect-derived values (0 = absent).
	achillesPermille int32 // Achilles or Aran High Defense x, job-selected
	manaReflectPct   int32

	// chakraPct is the WZ `x` of the caster's active Chakra recovery window
	// (0 = no window). CUserLocal::SetDamaged rewrites the raw damage by
	// this factor before every other term reads it (design §3.3), so it is
	// applied first, and it is deliberately NOT gated on the attack source —
	// there is no attackIdx, mob-sourced or magic/physical test around the
	// client's branch.
	//
	// x > 100 amplifies (GMS 12/48: 200..112); x < 100 reduces (GMS 61+:
	// 99..70; GMS 95: 96..60). The WZ data carries the direction, so there
	// is no version gate here and adding one would be the bug (design §4.2).
	chakraPct int32

	// Version gates, resolved from the tenant by the orchestrator.
	// Post-merge legacy verification (design §3) confirmed all three gates
	// hold across every column v48..jms with NO code change: the pre-BB
	// legacy versions (v48/61/72/79/84/92) all use pgCapDivisor 10,
	// fixedDamage min, and fall below the >=95 GUARD/Mechanic rule; v92 was
	// verified against its own IDB (NOT inherited from v87) and is
	// byte-identical to v83 here. (v48 additionally OMITS the fixedDamage
	// clamp / PG invincibility-zero / MR MaxHP/20 cap client-side; the
	// server applies all three universally since they only bound a reflect
	// downward — safe, so no v48-specific gate is added.)
	// magicShieldOnReducedDamage: MajorVersion >= 87 — base is damage
	// minus the Magic Guard portion (else base is raw damage, the v83
	// form). The gate is region-agnostic, so it also resolves true for
	// JMS 185; that >=87 form is a RETAINED design-phase finding (design
	// §3 / plan §7 finding 6), not one re-corroborated against the
	// legacy/JMS IDBs — the form is stat-cookie-driven and the legacy
	// immediate-search pass could not re-confirm it there. Treat this as
	// an unverified gate for JMS, not a verified one.
	magicShieldOnReducedDamage bool
	pgCapDivisor               int32 // 2 on GMS >= 95, else 10 (IDA-verified across all 10 columns)
	pgFixedDamageOverride      bool  // GMS >= 95 or JMS: template fixedDamage replaces the reflect instead of min()
}

type mobInfo struct {
	present     bool
	alive       bool
	maxHp       uint32
	boss        bool
	fixedDamage uint32
}

type reflectIntent struct {
	amount     uint32
	attackType byte
}

type mitigationBreakdown struct {
	chakraAmplified    int32
	achillesReduce     int32
	comboBarrierReduce int32
	magicShieldReduce  int32
	magicGuardAbsorbed int32
	mesoGuarded        int32
	powerGuardReflect  int32
}

type mitigationResult struct {
	hpLoss    int32
	mpLoss    int32
	mesoCost  int32
	reflect   reflectIntent // amount 0 = none
	breakdown mitigationBreakdown
}

// clampDamage bounds the client-supplied damage per FR-10.1. The -1 block
// sentinel is handled by the caller before clamping.
func clampDamage(raw int32) (int32, bool) {
	if raw < 0 {
		return 0, true
	}
	if raw > maxLegitimateDamage {
		return maxLegitimateDamage, true
	}
	return raw, false
}

// clampInt16 bounds an int32 delta to the CHANGE_HP/CHANGE_MP int16
// contract (FR-10.2 — replaces the silent int16 truncation).
func clampInt16(v int32) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}

// computeMitigation is the server mirror of the client's damage-taken
// math (design task-157 §6, IDA-verified v83/v87/v95/jms185). Integer
// arithmetic follows the decompiled formulas exactly.
func computeMitigation(in mitigationInput, mob mobInfo) mitigationResult {
	var r mitigationResult
	raw := in.rawDamage
	if raw <= 0 {
		return r
	}

	if in.chakraPct > 0 {
		raw = raw * in.chakraPct / 100
		// The client's floor is `<= 1 -> 1`, deliberately not `< 1`, and it
		// applies to the multiplied value rather than the original.
		if raw <= 1 {
			raw = 1
		}
		r.breakdown.chakraAmplified = raw
	}

	var achillesReduce int32
	if in.achillesPermille > 0 {
		achillesReduce = raw * (1000 - in.achillesPermille) / 1000
	}

	var comboBarrierReduce int32
	if in.comboBarrierPermille > 0 {
		comboBarrierReduce = (raw - achillesReduce) * (1000 - in.comboBarrierPermille) / 1000
	}

	var magicGuardPortion, mpLoss, absorbed int32
	if in.magicGuardPct > 0 {
		magicGuardPortion = raw * in.magicGuardPct / 100
		mpLoss = magicGuardPortion
		if mpLoss > int32(in.currentMP) {
			mpLoss = int32(in.currentMP)
		}
		absorbed = mpLoss
		if in.infinity {
			absorbed = magicGuardPortion
			mpLoss = 0
		}
	}

	var magicShieldReduce int32
	if in.magicShieldPct > 0 {
		base := raw
		if in.magicShieldOnReducedDamage {
			base = raw - magicGuardPortion
		}
		magicShieldReduce = base * in.magicShieldPct / 100
	}

	var mesoGuarded, mesoCost int32
	if in.mesoGuardPct > 0 && in.mobSourced {
		mesoGuarded = raw / 2
		cost := int64(in.mesoGuardPct) * int64(mesoGuarded) / 100
		if cost > int64(in.meso) {
			// Partial guard: scale the guarded share down to what the
			// meso balance affords (CalcDamage::GetMesoGuardReduce).
			mesoGuarded = int32(int64(100) * int64(in.meso) / int64(in.mesoGuardPct))
			cost = int64(in.mesoGuardPct) * int64(mesoGuarded) / 100
		}
		mesoCost = int32(cost)
	}

	var pgReflect int32
	if in.powerGuardSignal && in.powerGuardPct > 0 && in.mobSourced {
		if mob.present && mob.alive {
			pgReflect = in.powerGuardPct * raw / 100
			divisor := in.pgCapDivisor
			if divisor <= 0 {
				divisor = 10
			}
			reflectCap := int32(mob.maxHp / uint32(divisor))
			if pgReflect > reflectCap {
				pgReflect = reflectCap
			}
			if mob.boss {
				pgReflect /= 2
			}
			if pgReflect > 0 && mob.fixedDamage > 0 {
				fixed := int32(mob.fixedDamage)
				if in.pgFixedDamageOverride || fixed < pgReflect {
					pgReflect = fixed
				}
			}
		}
	}

	hpLoss := raw - achillesReduce - comboBarrierReduce - magicShieldReduce - absorbed - mesoGuarded - pgReflect
	if hpLoss < 0 {
		hpLoss = 0
	}

	r.hpLoss = hpLoss
	r.mpLoss = mpLoss
	r.mesoCost = mesoCost
	r.breakdown = mitigationBreakdown{
		chakraAmplified:    r.breakdown.chakraAmplified,
		achillesReduce:     achillesReduce,
		comboBarrierReduce: comboBarrierReduce,
		magicShieldReduce:  magicShieldReduce,
		magicGuardAbsorbed: absorbed,
		mesoGuarded:        mesoGuarded,
		powerGuardReflect:  pgReflect,
	}
	if pgReflect > 0 {
		r.reflect = reflectIntent{amount: uint32(pgReflect), attackType: reflectAttackTypePhysical}
	}

	if in.manaReflectSignal && in.manaReflectPct > 0 && in.mobSourced && mob.present && mob.alive {
		mr := raw * in.manaReflectPct / 100
		mrCap := int32(mob.maxHp / 20)
		if mr > mrCap {
			mr = mrCap
		}
		if mr > 0 {
			// Mana Reflection does not reduce the caster's own damage.
			r.reflect = reflectIntent{amount: uint32(mr), attackType: reflectAttackTypeMagic}
		}
	}

	return r
}
