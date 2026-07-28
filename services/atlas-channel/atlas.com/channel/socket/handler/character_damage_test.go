package handler

import (
	"math"
	"testing"
)

func TestGaugeCooldownValue(t *testing.T) {
	tests := []struct {
		name      string
		remaining int32
		expected  uint16
	}{
		{"normal", 8500, 8500},
		{"formula max fits (v87+ arm, SLV 10 @ 200)", 29000, 29000},
		{"defensive clamp above uint16", math.MaxUint16 + 1, math.MaxUint16},
		{"defensive floor below zero", -5, 0},
		{"one", 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := gaugeCooldownValue(tc.remaining); got != tc.expected {
				t.Errorf("gaugeCooldownValue(%d) = %d, want %d", tc.remaining, got, tc.expected)
			}
		})
	}
}
