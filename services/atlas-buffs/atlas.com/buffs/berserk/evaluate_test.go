package berserk

import "testing"

// v83 reference values (verified from local WZ, design §2): skill 1320006 has
// 30 levels, x = 21 at level 1 rising to 50 at level 30. Values here are test
// inputs only — runtime x always comes from atlas-data.
func TestEvaluate(t *testing.T) {
	cases := []struct {
		name       string
		skillLevel byte
		hp         uint16
		maxHp      uint32
		x          int16
		want       bool
	}{
		{name: "below threshold is active", skillLevel: 30, hp: 499, maxHp: 1000, x: 50, want: true},
		{name: "equality is inactive (strict less-than, Character.java:1852)", skillLevel: 30, hp: 500, maxHp: 1000, x: 50, want: false},
		{name: "above threshold is inactive", skillLevel: 30, hp: 501, maxHp: 1000, x: 50, want: false},
		{name: "integer division truncates toward inactive edge", skillLevel: 30, hp: 509, maxHp: 1020, x: 49, want: false}, // 509*100/1020 = 49
		{name: "integer division one below", skillLevel: 30, hp: 499, maxHp: 1020, x: 49, want: true}, // 499*100/1020 = 48
		{name: "skill level zero never active", skillLevel: 0, hp: 1, maxHp: 1000, x: 50, want: false},
		{name: "dead (hp=0) never active (design D7)", skillLevel: 30, hp: 0, maxHp: 1000, x: 50, want: false},
		{name: "maxHp zero guarded", skillLevel: 30, hp: 100, maxHp: 0, x: 50, want: false},
		{name: "non-positive x guarded", skillLevel: 30, hp: 1, maxHp: 1000, x: 0, want: false},
		{name: "negative x guarded", skillLevel: 30, hp: 1, maxHp: 1000, x: -1, want: false},
		{name: "hyper body raises maxHp and activates with constant hp", skillLevel: 30, hp: 600, maxHp: 1900, x: 50, want: true},   // 600*100/1900 = 31
		{name: "hyper body expiry deactivates with constant hp", skillLevel: 30, hp: 600, maxHp: 1000, x: 50, want: false},          // 60
		{name: "max uint16 hp does not overflow uint32 math", skillLevel: 30, hp: 65535, maxHp: 99999, x: 50, want: false},          // 65535*100 = 6,553,500 < 2^32
		{name: "level 1 threshold", skillLevel: 1, hp: 209, maxHp: 1000, x: 21, want: true},
		{name: "level 1 threshold boundary", skillLevel: 1, hp: 210, maxHp: 1000, x: 21, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Evaluate(tc.skillLevel, tc.hp, tc.maxHp, tc.x); got != tc.want {
				t.Errorf("Evaluate(%d, %d, %d, %d) = %v, want %v", tc.skillLevel, tc.hp, tc.maxHp, tc.x, got, tc.want)
			}
		})
	}
}
