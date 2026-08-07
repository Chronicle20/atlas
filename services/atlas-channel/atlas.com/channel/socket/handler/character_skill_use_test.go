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
