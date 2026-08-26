package monster

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// summaryOf indexes a DamageSummary by characterId for assertion.
func summaryOf(t *testing.T, m Model) map[uint32]uint32 {
	t.Helper()
	out := make(map[uint32]uint32)
	for _, e := range m.DamageSummary() {
		out[e.CharacterId] = e.Damage
	}
	return out
}

// TestDecayLeavesExperienceLedgerIntact pins the split: aggro decay may shrink
// and prune the aggro table, but the EXP ledger that feeds the DAMAGED/KILLED
// events must retain the full damage the character actually dealt.
//
// Regression: observed in atlas-pr-1456, where a 37000-HP monster emitted a
// KILLED event whose damage entries summed to 31932 because one contributor
// idled past AggroIdleThresholdMs before the kill.
func TestDecayLeavesExperienceLedgerIntact(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100000, 50, "", "")

	if _, err := r.ApplyDamage(ten, 1, 7906, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	// Decay the idle entry all the way to a prune.
	now := int64(20_000)
	var last DecaySummary
	for i := 0; i < 100; i++ {
		s, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
		if err != nil {
			t.Fatalf("Decay %d: %v", i, err)
		}
		last = s
		if len(s.Monster.DamageEntries()) == 0 {
			break
		}
	}

	if len(last.Monster.DamageEntries()) != 0 {
		t.Fatalf("aggro entries should have pruned, got %d", len(last.Monster.DamageEntries()))
	}

	got := summaryOf(t, last.Monster)
	if got[1] != 7906 {
		t.Errorf("experience ledger want 7906 for character 1, got %d (%+v)", got[1], last.Monster.DamageSummary())
	}
}

// TestClearAggroLeavesExperienceLedgerIntact pins the second half of the split:
// a CLEAR_AGGRO wipe empties the aggro table but must not erase EXP credit,
// which would cost the character both experience and quest kill progress.
func TestClearAggroLeavesExperienceLedgerIntact(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100000, 50, "", "")

	if _, err := r.ApplyDamage(ten, 1, 500, m.UniqueId(), 100); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 2, 250, m.UniqueId(), 100); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}

	s, err := r.ClearDamageEntries(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("ClearDamageEntries: %v", err)
	}

	if len(s.Monster.DamageEntries()) != 0 {
		t.Errorf("aggro table should be empty after clear, got %d", len(s.Monster.DamageEntries()))
	}

	got := summaryOf(t, s.Monster)
	if got[1] != 500 || got[2] != 250 {
		t.Errorf("experience ledger want {1:500, 2:250}, got %+v", got)
	}
}

// TestDamageLeaderUsesExperienceLedger verifies drop ownership is decided by
// damage actually dealt, not by the decayed aggro figure. Character 1 out-damages
// character 2 but goes idle; the aggro table would name character 2.
func TestDamageLeaderUsesExperienceLedger(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100000, 50, "", "")

	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 1000, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 2, 900, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}

	// Decay character 1's idle entry below character 2's fresh one.
	for i := 0; i < 3; i++ {
		if _, err := r.DecayDamageEntries(ten, m.UniqueId(), now); err != nil {
			t.Fatalf("Decay %d: %v", i, err)
		}
	}

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if leader := got.DamageLeader(); leader != 1 {
		t.Errorf("DamageLeader want 1 (dealt 1000), got %d; summary=%+v", leader, got.DamageSummary())
	}
}

// TestApplyDamageClampWritesBothLedgers verifies the clamped `actual` value --
// never the raw requested damage -- lands in the EXP ledger, so a killing blow
// that overkills cannot inflate a character's share past the monster's HP.
func TestApplyDamageClampWritesBothLedgers(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50, "", "")

	if _, err := r.ApplyDamage(ten, 1, 400, m.UniqueId(), 100); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	// Overkill: only 600 HP remains.
	s, err := r.ApplyDamage(ten, 2, 5000, m.UniqueId(), 100)
	if err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}

	got := summaryOf(t, s.Monster)
	if got[1] != 400 || got[2] != 600 {
		t.Errorf("experience ledger want {1:400, 2:600}, got %+v", got)
	}
}

// TestBuilderAddDamageEntryFeedsExperienceLedger keeps the builder write path
// in agreement with Registry.ApplyDamage: both must credit the EXP ledger.
func TestBuilderAddDamageEntryFeedsExperienceLedger(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := NewMonster(f, 1, 9300018, 0, 0, 0, 5, 0, 1000, 50, "", "")

	m = Clone(m).
		AddDamageEntry(1, 100).
		AddDamageEntry(1, 250).
		AddDamageEntry(2, 300).
		Build()

	got := make(map[uint32]uint32)
	for _, e := range m.DamageSummary() {
		got[e.CharacterId] = e.Damage
	}
	if got[1] != 350 || got[2] != 300 {
		t.Errorf("experience ledger want {1:350, 2:300}, got %+v", got)
	}
}
