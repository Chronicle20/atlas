package summon

import "testing"

func TestLookupKnownSummon(t *testing.T) {
	e, ok := Lookup(3111002) // Ranger Puppet
	if !ok || e.Type != TypePuppet || e.Movement != MovementStationary {
		t.Fatalf("ranger puppet wrong: %+v ok=%v", e, ok)
	}
	e, ok = Lookup(1321007) // Beholder
	if !ok || e.Type != TypeBuffAura || e.Movement != MovementFollow {
		t.Fatalf("beholder wrong: %+v ok=%v", e, ok)
	}
	e, ok = Lookup(3111005) // Silver Hawk: attacker, stun, circle-follow
	if !ok || e.Type != TypeAttacker || e.Movement != MovementCircleFollow || !e.Stun {
		t.Fatalf("silver hawk wrong: %+v ok=%v", e, ok)
	}
}

func TestLookupUnknownSummon(t *testing.T) {
	if _, ok := Lookup(99999999); ok {
		t.Fatalf("expected miss for unknown id")
	}
	if IsSummonSkill(99999999) {
		t.Fatalf("IsSummonSkill should be false for unknown id")
	}
	if !IsSummonSkill(1321007) {
		t.Fatalf("IsSummonSkill should be true for Beholder")
	}
}

func TestRosterHas21Entries(t *testing.T) {
	if len(roster) != 21 {
		t.Fatalf("expected 21 roster entries, got %d", len(roster))
	}
}
