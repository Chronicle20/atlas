package chakra

import (
	"math"
	"testing"
)

// TestCanActivateBoundary pins the client's gate (design §3.2):
// HP*100/MaxHP >= 50 rejects. The integer-equivalent form is 2*HP >= MaxHP.
// PRD OQ-9: exactly 50% must NOT activate.
func TestCanActivateBoundary(t *testing.T) {
	tests := []struct {
		name  string
		hp    uint16
		maxHp uint16
		want  bool
	}{
		{"49 percent", 49, 100, true},
		{"exactly 50 percent", 50, 100, false},
		{"51 percent", 51, 100, false},
		{"full hp", 100, 100, false},
		{"zero hp", 0, 100, true},
		{"odd maxhp just under half", 50, 101, true},
		{"odd maxhp at half rounded up", 51, 101, false},
		{"zero maxhp is never castable", 10, 0, false},
		{"max uint16 maxhp under half", 32767, 65535, true},
		{"max uint16 maxhp at half", 32768, 65535, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanActivate(tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("CanActivate(%d, %d) = %v, want %v", tc.hp, tc.maxHp, got, tc.want)
			}
		})
	}
}

// TestCanActivateMatchesClientForm sweeps the integer-equivalence claim
// against the literal client expression floor(HP*100/MaxHP) >= 50 so the
// rewrite cannot drift at an odd MaxHP.
func TestCanActivateMatchesClientForm(t *testing.T) {
	for maxHp := 1; maxHp <= 400; maxHp++ {
		for hp := 0; hp <= maxHp; hp++ {
			clientRejects := hp*100/maxHp >= 50
			if got := CanActivate(uint16(hp), uint16(maxHp)); got == clientRejects {
				t.Fatalf("CanActivate(%d, %d) = %v but the client form rejects = %v", hp, maxHp, got, clientRejects)
			}
		}
	}
}

// TestBase pins the community-sourced base recovery term: 2.9 x effective
// LUK, integer (design §3.4). Deliberately deterministic — no RNG.
func TestBase(t *testing.T) {
	tests := []struct {
		luck uint32
		want int32
	}{
		{0, 0},
		{1, 2},
		{10, 29},
		{100, 290},
		{123, 356},
		{math.MaxUint32, math.MaxInt32},
	}
	for _, tc := range tests {
		if got := Base(tc.luck); got != tc.want {
			t.Fatalf("Base(%d) = %d, want %d", tc.luck, got, tc.want)
		}
	}
}

// TestRecovery pins healAmount = base * y / 100 across the three distinct
// per-version y tables recorded in design §4.2.
func TestRecovery(t *testing.T) {
	tests := []struct {
		name string
		base int32
		y    int16
		want int32
	}{
		{"v48 L1 y=9", 290, 9, 26},
		{"v48 L30 y=200", 290, 200, 580},
		{"v83 L1 y=68", 290, 68, 197},
		{"v83 L30 y=300", 290, 300, 870},
		{"v95 L1 y=120", 290, 120, 348},
		{"v95 L10 y=300", 290, 300, 870},
		{"zero y", 290, 0, 0},
		{"negative y", 290, -5, 0},
		{"zero base", 0, 300, 0},
		{"negative base", -10, 300, 0},
		{"overflow guard", math.MaxInt32, 300, math.MaxInt32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Recovery(tc.base, tc.y); got != tc.want {
				t.Fatalf("Recovery(%d, %d) = %d, want %d", tc.base, tc.y, got, tc.want)
			}
		})
	}
}

// TestApplied pins the max-HP clamp (FR-3.2) and the never-negative rule
// (FR-3.5): Chakra never pushes HP past max and never applies a damage event.
func TestApplied(t *testing.T) {
	tests := []struct {
		name  string
		heal  int32
		hp    uint16
		maxHp uint16
		want  int16
	}{
		{"fits under cap", 100, 200, 1000, 100},
		{"clamped to missing", 5000, 900, 1000, 100},
		{"exactly missing", 100, 900, 1000, 100},
		{"at full hp", 500, 1000, 1000, 0},
		{"hp above max", 500, 1200, 1000, 0},
		{"zero heal", 0, 100, 1000, 0},
		{"negative heal", -50, 100, 1000, 0},
		{"int16 contract", math.MaxInt32, 0, 65535, math.MaxInt16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Applied(tc.heal, tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("Applied(%d, %d, %d) = %d, want %d", tc.heal, tc.hp, tc.maxHp, got, tc.want)
			}
		})
	}
}

// TestEffectiveMaxHpOrBase pins the defensive narrowing: a zero or
// out-of-range effective MaxHp falls back to the character record's base.
func TestEffectiveMaxHpOrBase(t *testing.T) {
	tests := []struct {
		effective uint32
		base      uint16
		want      uint16
	}{
		{0, 4000, 4000},
		{5000, 4000, 5000},
		{math.MaxUint16 + 1, 4000, math.MaxUint16},
		{math.MaxUint32, 4000, math.MaxUint16},
	}
	for _, tc := range tests {
		if got := EffectiveMaxHpOrBase(tc.effective, tc.base); got != tc.want {
			t.Fatalf("EffectiveMaxHpOrBase(%d, %d) = %d, want %d", tc.effective, tc.base, got, tc.want)
		}
	}
}
