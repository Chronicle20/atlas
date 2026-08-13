package handler

import (
	"testing"
	"time"
)

func TestBattleshipCastBlocked(t *testing.T) {
	now := time.Now()
	future := now.Add(30 * time.Second)
	past := now.Add(-30 * time.Second)
	tests := []struct {
		name              string
		skillId           uint32
		cooldownExpiresAt time.Time
		expected          bool
	}{
		{"battleship on cooldown blocked (FR-2.4)", 5221006, future, true},
		{"battleship cooldown expired allowed", 5221006, past, false},
		{"battleship never cooled allowed", 5221006, time.Time{}, false},
		{"other skill on cooldown not blocked here", 2311001, future, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := battleshipCastBlocked(tc.skillId, tc.cooldownExpiresAt, now); got != tc.expected {
				t.Errorf("battleshipCastBlocked(%d, %v) = %v, want %v", tc.skillId, tc.cooldownExpiresAt, got, tc.expected)
			}
		})
	}
}

// TestChakraUseBlocked pins PRD FR-1.4 / FR-5.4: a USE_SKILL for Chakra is
// honoured only when a recovery window is open. No window means the cast was
// never prepared (a crafted client skipping the prepare packet) or was
// interrupted — either way it is rejected BEFORE handler.UseSkill, so no MP
// is charged and no cooldown is applied.
//
// Deliberately NOT re-checking HP here: the client has no post-gate HP
// re-check (design §3.2) and PRD FR-1.3 requires the threshold be evaluated
// at activation only, so external healing mid-window must not cancel the
// heal. The window-presence check already closes the crafted-client hole a
// second HP check would have covered, and closes it more tightly.
func TestChakraUseBlocked(t *testing.T) {
	if chakraUseBlocked(true) {
		t.Fatal("an open recovery window must not block the cast")
	}
	if !chakraUseBlocked(false) {
		t.Fatal("a missing recovery window must block the cast")
	}
}
