package mprecovery

import (
	"math"
	"testing"
)

// TestAmounts pins the MP Recovery amount formula against WZ-verified v83
// values for 5101005 (x=10 at every level; y=55 at L1, 75 at L5, 100 at L10).
// Integer floor division at each step.
func TestAmounts(t *testing.T) {
	tests := []struct {
		name       string
		maxHp      uint16
		x          int16
		y          int16
		wantHpLost int16
		wantMpGain int16
	}{
		{name: "L1 v83 (x=10,y=55) maxHp=1234", maxHp: 1234, x: 10, y: 55, wantHpLost: 123, wantMpGain: 67},
		{name: "L5 v83 (x=10,y=75) maxHp=1234", maxHp: 1234, x: 10, y: 75, wantHpLost: 123, wantMpGain: 92},
		{name: "L10 v83 (x=10,y=100) maxHp=1234", maxHp: 1234, x: 10, y: 100, wantHpLost: 123, wantMpGain: 123},
		{name: "L10 v83 maxHp=30000", maxHp: 30000, x: 10, y: 100, wantHpLost: 3000, wantMpGain: 3000},
		{name: "floor on mpGain (hpLost*y not divisible)", maxHp: 100, x: 10, y: 55, wantHpLost: 10, wantMpGain: 5},
		{name: "maxHp below x floors hpLost to zero", maxHp: 9, x: 10, y: 100, wantHpLost: 0, wantMpGain: 0},
		{name: "x=0 returns zeros (bad tenant data)", maxHp: 1234, x: 0, y: 55, wantHpLost: 0, wantMpGain: 0},
		{name: "x negative returns zeros", maxHp: 1234, x: -5, y: 55, wantHpLost: 0, wantMpGain: 0},
		{name: "pathological x=1 at uint16 max clamps, never wraps", maxHp: math.MaxUint16, x: 1, y: 100, wantHpLost: math.MaxInt16, wantMpGain: math.MaxInt16},
		{name: "negative y floors mpGain at zero", maxHp: 1234, x: 10, y: -50, wantHpLost: 123, wantMpGain: 0},
		{name: "mpGain from unclamped hpLost (x=1) not post-clamp delta", maxHp: math.MaxUint16, x: 1, y: 50, wantHpLost: math.MaxInt16, wantMpGain: math.MaxInt16},
		{name: "mpGain-only clamp: hpLost in range, mpGain overflows", maxHp: math.MaxUint16, x: 3, y: math.MaxInt16, wantHpLost: 21845, wantMpGain: math.MaxInt16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHp, gotMp := Amounts(tc.maxHp, tc.x, tc.y)
			if gotHp != tc.wantHpLost || gotMp != tc.wantMpGain {
				t.Fatalf("Amounts(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tc.maxHp, tc.x, tc.y, gotHp, gotMp, tc.wantHpLost, tc.wantMpGain)
			}
		})
	}
}
