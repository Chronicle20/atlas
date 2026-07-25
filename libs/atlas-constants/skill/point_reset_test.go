package skill_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsPointResetExcluded(t *testing.T) {
	excluded := []skill.Id{
		21110007, 21110008, 21120009, 21120010, // Aran hidden combo skills
		9001000, 9101008, 9050000, // GM skill range bounds + interior
		8001000, 8001001, // GM skills
		20000014, 20000018, // PQ skill range bounds
		10000013, 20001013, // PQ skills (fixed ids)
		1009, 1010, 1011, 10001009, 20001011, // id%10000000 in 1009-1011
		1020, 20001020, // id%10000000 == 1020
	}
	for _, id := range excluded {
		if !skill.IsPointResetExcluded(id) {
			t.Errorf("IsPointResetExcluded(%d) = false, want true", id)
		}
	}
	included := []skill.Id{
		1001003,    // Iron Body (1st job warrior)
		3121004,    // Hurricane (4th job bowman)
		2301002,    // Heal
		21100000,   // Aran non-hidden
		1012, 1008, // just outside the 1009-1011 band
	}
	for _, id := range included {
		if skill.IsPointResetExcluded(id) {
			t.Errorf("IsPointResetExcluded(%d) = true, want false", id)
		}
	}
}
