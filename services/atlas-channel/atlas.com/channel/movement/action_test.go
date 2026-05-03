package movement

import "testing"

func TestBasicAttackPos_OutOfRange(t *testing.T) {
	cases := []int8{-1, 0, 23, 42, 60, 100}
	for _, c := range cases {
		if pos, ok := basicAttackPos(c); ok {
			t.Errorf("basicAttackPos(%d) = (%d, true), want (_, false)", c, pos)
		}
	}
}

func TestBasicAttackPos_InRange(t *testing.T) {
	cases := map[int8]uint8{
		24: 0, 25: 0,
		26: 1, 27: 1,
		28: 2, 29: 2,
		40: 8, 41: 8,
	}
	for raw, want := range cases {
		got, ok := basicAttackPos(raw)
		if !ok {
			t.Errorf("basicAttackPos(%d) = (_, false), want (%d, true)", raw, want)
			continue
		}
		if got != want {
			t.Errorf("basicAttackPos(%d) = %d, want %d", raw, got, want)
		}
	}
}
