package summon

import "testing"

func TestConservativeCeilingClampsExcess(t *testing.T) {
	// physical summon: watk=200, effect.weaponAttack=100
	// reported damage way above the bound is clamped; in-bound damage passes through.
	max := ConservativeMaxPerHit(false /*magic*/, 200 /*watk*/, 0 /*matk*/, 100 /*effWatk*/, 0 /*effMatk*/)
	if clampDamage(uint32(max)+5000, max) != uint32(max) {
		t.Fatalf("excess not clamped")
	}
	if clampDamage(uint32(max)-1, max) != uint32(max)-1 {
		t.Fatalf("in-bound damage altered")
	}
}
