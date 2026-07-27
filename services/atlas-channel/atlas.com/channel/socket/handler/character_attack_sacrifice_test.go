package handler

import (
	"math"
	"testing"
)

func TestSacrificeHpCost(t *testing.T) {
	cases := []struct {
		name      string
		firstLine uint32
		x         int16
		currentHp uint16
		want      uint16
	}{
		{"normal computation", 1000, 30, 5000, 300},
		{"truncating division", 99, 30, 5000, 29},
		{"x zero", 1000, 0, 5000, 0},
		{"x negative", 1000, -5, 5000, 0},
		{"miss (first line zero)", 0, 30, 5000, 0},
		{"clamp to hp minus one", 100000, 100, 500, 499},
		{"exact-kill boundary clamps", 1000, 100, 1000, 999},
		{"hp one is a no-op", 1000, 30, 1, 0},
		{"hp zero is a no-op", 1000, 30, 0, 0},
		{"narrowing guard caps at MaxInt16", 100000, 100, 65535, math.MaxInt16},
		{"max uint32 line does not wrap", math.MaxUint32, 100, 30000, 29999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sacrificeHpCost(tc.firstLine, tc.x, tc.currentHp); got != tc.want {
				t.Fatalf("sacrificeHpCost(%d, %d, %d) = %d; want %d", tc.firstLine, tc.x, tc.currentHp, got, tc.want)
			}
		})
	}
}
