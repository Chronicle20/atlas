package handler

import (
	"math"
	"testing"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func mobUp(maxHp uint32) mobInfo {
	return mobInfo{present: true, alive: true, maxHp: maxHp}
}

func TestComputeMitigationNoBuffPassthrough(t *testing.T) {
	in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 500, mobSourced: true, pgCapDivisor: 10}
	r := computeMitigation(in, mobUp(1000))
	if r.hpLoss != 500 || r.mpLoss != 0 || r.mesoCost != 0 || r.reflect.amount != 0 {
		t.Fatalf("passthrough broken: %+v", r)
	}
}

func TestComputeMitigationMagicGuard(t *testing.T) {
	// Magic Guard lvl 20: x=80 (v83 Skill.wz 2001002).
	base := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, pgCapDivisor: 10}

	t.Run("standard split", func(t *testing.T) {
		in := base
		in.currentMP = 5000
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 800 || r.hpLoss != 200 {
			t.Fatalf("mp=%d hp=%d, want 800/200", r.mpLoss, r.hpLoss)
		}
	})
	t.Run("MP shortfall spills to HP", func(t *testing.T) {
		in := base
		in.currentMP = 300
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 300 || r.hpLoss != 700 {
			t.Fatalf("mp=%d hp=%d, want 300/700", r.mpLoss, r.hpLoss)
		}
	})
	t.Run("Infinity absorbs fully with no MP cost", func(t *testing.T) {
		in := base
		in.currentMP = 0
		in.infinity = true
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 0 || r.hpLoss != 200 {
			t.Fatalf("mp=%d hp=%d, want 0/200", r.mpLoss, r.hpLoss)
		}
	})
}

func TestComputeMitigationMesoGuard(t *testing.T) {
	// Meso Guard: x is the meso cost rate (v83 4211005: 81-90).
	base := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, mesoGuardPct: 81, pgCapDivisor: 10}

	t.Run("standard half guard", func(t *testing.T) {
		in := base
		in.meso = 1000000
		r := computeMitigation(in, mobUp(1000))
		// guarded = 500, cost = 81*500/100 = 405
		if r.hpLoss != 500 || r.mesoCost != 405 {
			t.Fatalf("hp=%d cost=%d, want 500/405", r.hpLoss, r.mesoCost)
		}
	})
	t.Run("partial guard when meso short", func(t *testing.T) {
		in := base
		in.meso = 100
		r := computeMitigation(in, mobUp(1000))
		// guarded = 100*100/81 = 123, cost = 81*123/100 = 99
		if r.breakdown.mesoGuarded != 123 || r.mesoCost != 99 || r.hpLoss != 877 {
			t.Fatalf("guarded=%d cost=%d hp=%d, want 123/99/877", r.breakdown.mesoGuarded, r.mesoCost, r.hpLoss)
		}
	})
	t.Run("zero meso guards nothing", func(t *testing.T) {
		in := base
		in.meso = 0
		r := computeMitigation(in, mobUp(1000))
		if r.mesoCost != 0 || r.hpLoss != 1000 {
			t.Fatalf("cost=%d hp=%d, want 0/1000", r.mesoCost, r.hpLoss)
		}
	})
	t.Run("not applied to obstacle damage", func(t *testing.T) {
		in := base
		in.meso = 1000000
		in.mobSourced = false
		in.attackIdx = packetmodel.DamageTypeObstacle
		r := computeMitigation(in, mobInfo{})
		if r.mesoCost != 0 || r.hpLoss != 1000 {
			t.Fatalf("cost=%d hp=%d, want 0/1000", r.mesoCost, r.hpLoss)
		}
	})
}

func TestComputeMitigationPowerGuard(t *testing.T) {
	base := mitigationInput{
		attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true,
		powerGuardSignal: true, powerGuardPct: 30, pgCapDivisor: 10,
	}

	t.Run("reflect reduces own HP loss", func(t *testing.T) {
		r := computeMitigation(base, mobUp(100000))
		// reflect = 30*1000/100 = 300, cap = 100000/10 = 10000
		if r.reflect.amount != 300 || r.reflect.attackType != reflectAttackTypePhysical || r.hpLoss != 700 {
			t.Fatalf("reflect=%+v hp=%d, want 300/physical/700", r.reflect, r.hpLoss)
		}
	})
	t.Run("cap binds at maxHp/divisor", func(t *testing.T) {
		r := computeMitigation(base, mobUp(1000))
		// cap = 1000/10 = 100
		if r.reflect.amount != 100 || r.hpLoss != 900 {
			t.Fatalf("reflect=%d hp=%d, want 100/900", r.reflect.amount, r.hpLoss)
		}
	})
	t.Run("v95 divisor 2", func(t *testing.T) {
		in := base
		in.pgCapDivisor = 2
		r := computeMitigation(in, mobUp(1000))
		// cap = 1000/2 = 500, reflect = 300 uncapped
		if r.reflect.amount != 300 {
			t.Fatalf("reflect=%d, want 300", r.reflect.amount)
		}
	})
	t.Run("boss halves after cap", func(t *testing.T) {
		mob := mobUp(1000)
		mob.boss = true
		r := computeMitigation(base, mob)
		if r.reflect.amount != 50 {
			t.Fatalf("reflect=%d, want 50", r.reflect.amount)
		}
	})
	t.Run("fixedDamage min pre-BB", func(t *testing.T) {
		mob := mobUp(100000)
		mob.fixedDamage = 5
		r := computeMitigation(base, mob)
		if r.reflect.amount != 5 {
			t.Fatalf("reflect=%d, want 5", r.reflect.amount)
		}
	})
	t.Run("fixedDamage override post-BB", func(t *testing.T) {
		in := base
		in.pgFixedDamageOverride = true
		mob := mobUp(100000)
		mob.fixedDamage = 100000
		r := computeMitigation(in, mob)
		if r.reflect.amount != 100000 {
			t.Fatalf("reflect=%d, want 100000 (override, not min)", r.reflect.amount)
		}
	})
	t.Run("dead mob drops reflect but keeps HP application", func(t *testing.T) {
		mob := mobUp(1000)
		mob.alive = false
		r := computeMitigation(base, mob)
		if r.reflect.amount != 0 || r.hpLoss != 1000 {
			t.Fatalf("reflect=%d hp=%d, want 0/1000", r.reflect.amount, r.hpLoss)
		}
	})
	t.Run("no signal means no reflect regardless of buff", func(t *testing.T) {
		in := base
		in.powerGuardSignal = false
		r := computeMitigation(in, mobUp(100000))
		if r.reflect.amount != 0 || r.hpLoss != 1000 {
			t.Fatalf("reflect=%d hp=%d, want 0/1000", r.reflect.amount, r.hpLoss)
		}
	})
}

func TestComputeMitigationManaReflection(t *testing.T) {
	in := mitigationInput{
		attackIdx: packetmodel.DamageTypeMagic, rawDamage: 1000, mobSourced: true,
		manaReflectSignal: true, manaReflectPct: 140, pgCapDivisor: 10,
	}
	t.Run("reflects without reducing own damage", func(t *testing.T) {
		r := computeMitigation(in, mobUp(100000))
		// reflect = 140*1000/100 = 1400, cap = 100000/20 = 5000
		if r.reflect.amount != 1400 || r.reflect.attackType != reflectAttackTypeMagic {
			t.Fatalf("reflect=%+v, want 1400/magic", r.reflect)
		}
		if r.hpLoss != 1000 {
			t.Fatalf("hp=%d, want 1000 (MR must not self-reduce)", r.hpLoss)
		}
	})
	t.Run("cap at maxHp/20", func(t *testing.T) {
		r := computeMitigation(in, mobUp(10000))
		if r.reflect.amount != 500 {
			t.Fatalf("reflect=%d, want 500", r.reflect.amount)
		}
	})
}

func TestComputeMitigationPassivesAndBarriers(t *testing.T) {
	t.Run("Achilles level 30", func(t *testing.T) {
		// x=850 per-mille -> reduce = 1000*(1000-850)/1000 = 150
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, achillesPermille: 850, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		if r.breakdown.achillesReduce != 150 || r.hpLoss != 850 {
			t.Fatalf("reduce=%d hp=%d, want 150/850", r.breakdown.achillesReduce, r.hpLoss)
		}
	})
	t.Run("Achilles applies to obstacle damage", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypeObstacle, rawDamage: 1000, mobSourced: false, achillesPermille: 850}
		r := computeMitigation(in, mobInfo{})
		if r.hpLoss != 850 {
			t.Fatalf("hp=%d, want 850", r.hpLoss)
		}
	})
	t.Run("Combo Barrier stacks after Achilles", func(t *testing.T) {
		// cb x=864: reduce2 = (1000-150)*(1000-864)/1000 = 115
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, achillesPermille: 850, comboBarrierPermille: 864, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		if r.breakdown.comboBarrierReduce != 115 || r.hpLoss != 735 {
			t.Fatalf("cb=%d hp=%d, want 115/735", r.breakdown.comboBarrierReduce, r.hpLoss)
		}
	})
	t.Run("Magic Shield v83 form uses raw damage", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, currentMP: 5000, magicShieldPct: 20, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		// ms = 1000*20/100 = 200; hp = 1000 - 200 - 800 = 0
		if r.breakdown.magicShieldReduce != 200 || r.hpLoss != 0 {
			t.Fatalf("ms=%d hp=%d, want 200/0", r.breakdown.magicShieldReduce, r.hpLoss)
		}
	})
	t.Run("Magic Shield >=87 form uses damage minus MG portion", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, currentMP: 5000, magicShieldPct: 20, magicShieldOnReducedDamage: true, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		// ms = (1000-800)*20/100 = 40; hp = 1000 - 40 - 800 = 160
		if r.breakdown.magicShieldReduce != 40 || r.hpLoss != 160 {
			t.Fatalf("ms=%d hp=%d, want 40/160", r.breakdown.magicShieldReduce, r.hpLoss)
		}
	})
}

func TestClampDamage(t *testing.T) {
	if v, adj := clampDamage(500); v != 500 || adj {
		t.Fatalf("got %d/%t", v, adj)
	}
	if v, adj := clampDamage(-5); v != 0 || !adj {
		t.Fatalf("forged negative: got %d/%t, want 0/true", v, adj)
	}
	if v, adj := clampDamage(50000000); v != maxLegitimateDamage || !adj {
		t.Fatalf("forged oversized: got %d/%t, want %d/true", v, adj, maxLegitimateDamage)
	}
}

func TestClampInt16(t *testing.T) {
	if clampInt16(40000) != math.MaxInt16 {
		t.Fatal("high clamp failed")
	}
	if clampInt16(-40000) != math.MinInt16 {
		t.Fatal("low clamp failed")
	}
	if clampInt16(-500) != -500 {
		t.Fatal("identity failed")
	}
}

// TestChakraFactor pins design §3.1: the client rewrites the raw damage by
// the caster's Chakra level `x` percent, with a <= 1 -> 1 floor, and does so
// with NO gate on the attack source. On GMS 12/48 x is 200..112 so the term
// AMPLIFIES; on GMS 61+ x is 99..70 so it REDUCES; on GMS 95 x is 96..60.
// The WZ data carries the direction — there is deliberately no version gate.
func TestChakraFactor(t *testing.T) {
	tests := []struct {
		name      string
		raw       int32
		chakraPct int32
		wantHp    int32
	}{
		{"no window", 500, 0, 500},
		{"v48 L1 x=200 amplifies", 500, 200, 1000},
		{"v48 L30 x=112 amplifies", 500, 112, 560},
		{"v83 L1 x=99 reduces", 500, 99, 495},
		{"v83 L30 x=70 reduces", 500, 70, 350},
		{"v95 L10 x=60 reduces", 500, 60, 300},
		{"floor at one", 1, 60, 1},
		{"floor applies to the product not the input", 2, 50, 1},
		{"rounding truncates", 7, 70, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeMitigation(mitigationInput{rawDamage: tc.raw, chakraPct: tc.chakraPct}, mobInfo{})
			if got.hpLoss != tc.wantHp {
				t.Fatalf("hpLoss = %d, want %d", got.hpLoss, tc.wantHp)
			}
		})
	}
}

// TestChakraFactorAppliedBeforeEveryOtherTerm pins design §3.3 / PRD FR-4.3:
// in CUserLocal::SetDamaged the Chakra branch writes back to the same stack
// slot that carries the damage, and every task-157 term reads that slot
// afterwards. Applying it after Achilles would produce different numbers.
func TestChakraFactorAppliedBeforeEveryOtherTerm(t *testing.T) {
	in := mitigationInput{
		rawDamage:            1000,
		chakraPct:            200,
		achillesPermille:     200,
		comboBarrierPermille: 100,
		magicGuardPct:        50,
		currentMP:            5000,
		mobSourced:           true,
	}
	got := computeMitigation(in, mobInfo{})

	// Same chain, with the Chakra factor already folded into rawDamage and
	// the term disabled — the result must be identical.
	pre := in
	pre.rawDamage = 2000
	pre.chakraPct = 0
	want := computeMitigation(pre, mobInfo{})

	if got.hpLoss != want.hpLoss || got.mpLoss != want.mpLoss {
		t.Fatalf("(hpLoss,mpLoss) = (%d,%d), want (%d,%d) — Chakra must be applied to raw damage before every other term",
			got.hpLoss, got.mpLoss, want.hpLoss, want.mpLoss)
	}
	if got.breakdown.achillesReduce == want.breakdown.achillesReduce && got.breakdown.achillesReduce == 0 {
		t.Fatal("test is not exercising Achilles")
	}
}

// TestChakraBreakdownReportsPostFactorDamage pins the observability
// requirement: without the post-factor value in the breakdown, "Chakra did
// nothing" is undiagnosable from logs.
func TestChakraBreakdownReportsPostFactorDamage(t *testing.T) {
	got := computeMitigation(mitigationInput{rawDamage: 500, chakraPct: 200}, mobInfo{})
	if got.breakdown.chakraAmplified != 1000 {
		t.Fatalf("breakdown.chakraAmplified = %d, want 1000", got.breakdown.chakraAmplified)
	}
	none := computeMitigation(mitigationInput{rawDamage: 500}, mobInfo{})
	if none.breakdown.chakraAmplified != 0 {
		t.Fatalf("breakdown.chakraAmplified = %d with no window, want 0", none.breakdown.chakraAmplified)
	}
}
