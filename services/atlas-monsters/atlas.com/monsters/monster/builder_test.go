package monster

import (
	"testing"
	"time"
)

func emptyBuilder() *ModelBuilder {
	return Clone(NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50, "", ""))
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

	first := mkVenomEffect(30 * time.Second) // earliest expiry
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

// TestAddDamageEntry_AggregatesByCharacter verifies that AddDamageEntry sums
// damage into the existing entry for a repeat characterId rather than
// appending a new one, and that entries appear in first-contact order.
func TestAddDamageEntry_AggregatesByCharacter(t *testing.T) {
	tests := []struct {
		name  string
		calls [][2]uint32 // characterId, damage
		want  []entry
	}{
		{
			name:  "same character sums",
			calls: [][2]uint32{{1, 100}, {1, 100}, {1, 100}},
			want:  []entry{{CharacterId: 1, Damage: 300, LastHitMs: 0}},
		},
		{
			name:  "two characters keep first-contact order",
			calls: [][2]uint32{{2, 50}, {1, 10}, {2, 25}},
			want: []entry{
				{CharacterId: 2, Damage: 75, LastHitMs: 0},
				{CharacterId: 1, Damage: 10, LastHitMs: 0},
			},
		},
		{
			name:  "single entry unchanged",
			calls: [][2]uint32{{7, 42}},
			want:  []entry{{CharacterId: 7, Damage: 42, LastHitMs: 0}},
		},
		{
			name:  "zero damage still creates an entry",
			calls: [][2]uint32{{3, 0}},
			want:  []entry{{CharacterId: 3, Damage: 0, LastHitMs: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := emptyBuilder()
			for _, c := range tt.calls {
				b.AddDamageEntry(c[0], c[1])
			}

			if len(b.damageEntries) != len(tt.want) {
				t.Fatalf("expected %d entries, got %d: %+v", len(tt.want), len(b.damageEntries), b.damageEntries)
			}
			for i, want := range tt.want {
				got := b.damageEntries[i]
				if got.CharacterId != want.CharacterId {
					t.Errorf("entry[%d].CharacterId = %d, want %d", i, got.CharacterId, want.CharacterId)
				}
				if got.Damage != want.Damage {
					t.Errorf("entry[%d].Damage = %d, want %d", i, got.Damage, want.Damage)
				}
				if got.LastHitMs != want.LastHitMs {
					t.Errorf("entry[%d].LastHitMs = %d, want %d", i, got.LastHitMs, want.LastHitMs)
				}
			}
		})
	}
}

// TestAddDamageEntry_PreservesExistingLastHitMs verifies that aggregating into
// an existing entry does not modify that entry's LastHitMs, since
// AddDamageEntry's signature carries no timestamp to update it with.
func TestAddDamageEntry_PreservesExistingLastHitMs(t *testing.T) {
	b := emptyBuilder()
	b.damageEntries = []entry{{CharacterId: 5, Damage: 100, LastHitMs: 900}}

	b.AddDamageEntry(5, 50)

	if len(b.damageEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(b.damageEntries), b.damageEntries)
	}
	got := b.damageEntries[0]
	want := entry{CharacterId: 5, Damage: 150, LastHitMs: 900}
	if got != want {
		t.Errorf("entry = %+v, want %+v", got, want)
	}
}

// TestDamageLeader_OverBuilderAggregatedEntries is a PRD acceptance check: the
// damage leader must be determined from aggregated per-character totals, not
// per-hit entries. A per-hit append would leave character 2's single 250-damage
// hit ranked above character 1's three 100-damage hits (250 > 100), when the
// correct leader is character 1 (300 > 250).
func TestDamageLeader_OverBuilderAggregatedEntries(t *testing.T) {
	b := emptyBuilder()
	b.AddDamageEntry(1, 100).AddDamageEntry(1, 100).AddDamageEntry(1, 100)
	b.AddDamageEntry(2, 250)

	m := b.Build()

	if got := len(m.DamageSummary()); got != 2 {
		t.Fatalf("expected 2 aggregated damage entries, got %d", got)
	}
	if got := m.DamageLeader(); got != 1 {
		t.Errorf("DamageLeader() = %d, want 1", got)
	}
}
