package monster

import (
	"testing"
	"time"
)

func emptyBuilder() *ModelBuilder {
	return Clone(NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50))
}

func mkVenomEffect(duration time.Duration) StatusEffect {
	return NewStatusEffect(
		SourceTypePlayerSkill, 1, 0, 0,
		map[string]int32{"VENOM": 100},
		duration,
		0,
	)
}

func countVenom(effs []StatusEffect) int {
	c := 0
	for _, e := range effs {
		if e.HasStatus("VENOM") {
			c++
		}
	}
	return c
}

func hasEffectWithExpiry(effs []StatusEffect, at time.Time) bool {
	for _, e := range effs {
		if e.ExpiresAt().Equal(at) {
			return true
		}
	}
	return false
}

// TestAddStatusEffect_VenomOverflow_EvictsByEarliestExpiresAt verifies that
// when adding a 4th VENOM effect, the effect with the earliest ExpiresAt is
// removed (not the first-inserted one). Insertion order is deliberately
// scrambled vs expiry order so a FIFO eviction would surface as a failure.
func TestAddStatusEffect_VenomOverflow_EvictsByEarliestExpiresAt(t *testing.T) {
	b := emptyBuilder()

	first := mkVenomEffect(30 * time.Second)   // earliest expiry
	second := mkVenomEffect(60 * time.Second)
	third := mkVenomEffect(90 * time.Second)
	fourth := mkVenomEffect(120 * time.Second)

	// Scrambled insertion order vs expiry order: second, third, first, fourth.
	// FIFO eviction would remove `second` (first inserted), but we want `first`
	// (earliest expiry) removed instead.
	b.AddStatusEffect(second).AddStatusEffect(third).AddStatusEffect(first).AddStatusEffect(fourth)

	if got := countVenom(b.statusEffects); got != 3 {
		t.Fatalf("expected VENOM cap=3 after overflow apply; got %d", got)
	}
	if hasEffectWithExpiry(b.statusEffects, first.ExpiresAt()) {
		t.Errorf("expected earliest-expiry effect to be evicted; first.ExpiresAt() still present")
	}
	if !hasEffectWithExpiry(b.statusEffects, second.ExpiresAt()) {
		t.Errorf("expected `second` to remain")
	}
	if !hasEffectWithExpiry(b.statusEffects, third.ExpiresAt()) {
		t.Errorf("expected `third` to remain")
	}
	if !hasEffectWithExpiry(b.statusEffects, fourth.ExpiresAt()) {
		t.Errorf("expected `fourth` (newly added) to remain")
	}
}

// TestAddStatusEffect_VenomConcurrentApplies_NeverExceedsThree verifies that
// repeated VENOM applies always respect the cap of 3 stacks.
func TestAddStatusEffect_VenomConcurrentApplies_NeverExceedsThree(t *testing.T) {
	b := emptyBuilder()
	for i := 0; i < 100; i++ {
		eff := NewStatusEffect(
			SourceTypePlayerSkill, 1, 0, 0,
			map[string]int32{"VENOM": int32(i)},
			time.Duration(i)*time.Second,
			0,
		)
		b.AddStatusEffect(eff)
	}
	if got := countVenom(b.statusEffects); got != 3 {
		t.Fatalf("expected VENOM cap=3 after 100 applies; got %d", got)
	}
}
