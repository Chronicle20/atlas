package handler

import (
	"math"
	"testing"
)

func TestMpEaterShouldProc(t *testing.T) {
	cases := []struct {
		name string
		prop float64
		roll float64
		want bool
	}{
		{"prop 1.0 always true", 1.0, 0.99, true},
		{"prop 1.0 with zero roll", 1.0, 0.0, true},
		{"prop 0.5 roll under", 0.5, 0.49, true},
		{"prop 0.5 roll equal", 0.5, 0.50, false},
		{"prop 0.5 roll over", 0.5, 0.51, false},
		{"prop 0.0 never", 0.0, 0.0, false},
		{"negative prop never", -1.0, 0.0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mpEaterShouldProc(tc.prop, tc.roll); got != tc.want {
				t.Fatalf("mpEaterShouldProc(%v, %v) = %v; want %v", tc.prop, tc.roll, got, tc.want)
			}
		})
	}
}

func TestMpEaterAbsorbAmount(t *testing.T) {
	cases := []struct {
		name  string
		maxMp uint32
		x     int16
		want  uint32
	}{
		{"normal", 1000, 10, 100},
		{"zero MaxMp", 0, 10, 0},
		{"zero X", 1000, 0, 0},
		{"negative X", 1000, -1, 0},
		{"large MaxMp does not overflow", math.MaxUint32, 10, math.MaxUint32 / 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mpEaterAbsorbAmount(tc.maxMp, tc.x); got != tc.want {
				t.Fatalf("mpEaterAbsorbAmount(%d, %d) = %d; want %d", tc.maxMp, tc.x, got, tc.want)
			}
		})
	}
}
