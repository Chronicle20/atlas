package monster

import (
	"reflect"
	"testing"
)

func TestIntervalSet_Contains(t *testing.T) {
	tests := []struct {
		name   string
		adds   [][2]int
		probes map[int]bool
	}{
		{
			name: "single interval",
			adds: [][2]int{{25, 35}},
			probes: map[int]bool{
				24: false,
				25: true,
				30: true,
				35: true,
				36: false,
			},
		},
		{
			name: "PRD worked example",
			adds: [][2]int{{120, 130}, {25, 35}, {115, 125}},
			probes: map[int]bool{
				32:  true,
				70:  false,
				25:  true,
				35:  true,
				115: true,
				130: true,
				36:  false,
				114: false,
			},
		},
		{
			name: "adjacent intervals merge",
			adds: [][2]int{{0, 10}, {11, 20}},
			probes: map[int]bool{
				10: true,
				11: true,
				20: true,
				21: false,
			},
		},
		{
			name: "overlapping intervals merge",
			adds: [][2]int{{0, 10}, {5, 20}},
			probes: map[int]bool{
				0:  true,
				15: true,
				20: true,
				21: false,
			},
		},
		{
			name: "disjoint intervals stay disjoint",
			adds: [][2]int{{0, 5}, {100, 105}},
			probes: map[int]bool{
				5:   true,
				6:   false,
				99:  false,
				100: true,
			},
		},
		{
			name: "negative lo clamps to zero",
			adds: [][2]int{{-5, 3}},
			probes: map[int]bool{
				0: true,
				3: true,
				4: false,
			},
		},
		{
			name: "unsorted adds",
			adds: [][2]int{{100, 110}, {0, 10}, {50, 60}},
			probes: map[int]bool{
				5:   true,
				55:  true,
				105: true,
				30:  false,
				70:  false,
			},
		},
		{
			name:   "empty set contains nothing",
			adds:   nil,
			probes: map[int]bool{0: false, 50: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s intervalSet
			for _, iv := range tt.adds {
				s.add(iv[0], iv[1])
			}
			built := s.build()
			for v, want := range tt.probes {
				if got := built.contains(v); got != want {
					t.Errorf("contains(%d) = %v, want %v", v, got, want)
				}
			}
		})
	}
}

func TestIntervalSet_BuildMergesRanges(t *testing.T) {
	var s intervalSet
	s.add(0, 10)
	s.add(5, 20)
	s.add(100, 105)

	built := s.build()

	if len(built.ivs) != 2 {
		t.Fatalf("len(built.ivs) = %d, want 2", len(built.ivs))
	}
	want := []interval{{lo: 0, hi: 20}, {lo: 100, hi: 105}}
	if !reflect.DeepEqual(built.ivs, want) {
		t.Errorf("built.ivs = %+v, want %+v", built.ivs, want)
	}
}

func TestIntervalSet_BuildIsIdempotent(t *testing.T) {
	var s intervalSet
	s.add(0, 10)
	s.add(5, 20)
	s.add(100, 105)

	built := s.build()
	rebuilt := built.build()

	if !reflect.DeepEqual(built, rebuilt) {
		t.Errorf("build() is not idempotent: built = %+v, rebuilt = %+v", built, rebuilt)
	}
}
