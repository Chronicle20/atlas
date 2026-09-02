package job

import (
	"fmt"
	"testing"
)

// The 3000-3999 arm (job/1000 == 3) is pinned per FR-7 even though no Atlas
// job identity lands in that band today — the only 3xx identities are Bowman
// 300 .. Marksman 322, all of which are < 1000 after the divide. Pinning it
// means a future Resistance bring-up inherits the client's rule instead of
// rediscovering it.

// FR-10: a Dual Blade is NOT an extended-SP job (430/1000 == 0). It takes the
// plain SP short while still taking the per-skill master-level int, so the two
// task-275 fixes must not be conflated.
func TestUsesExtendedSP(t *testing.T) {
	for _, c := range []struct {
		name  string
		jobId Id
		want  [6]bool
	}{
		{"2001 (Evan beginner)", 2001, [6]bool{false, true, true, true, true, true}},
		{"2200", 2200, [6]bool{false, true, true, true, true, true}},
		{"2210", 2210, [6]bool{false, true, true, true, true, true}},
		{"2218", 2218, [6]bool{false, true, true, true, true, true}},
		{"2299", 2299, [6]bool{false, true, true, true, true, true}},
		{"3000 (Resistance band)", 3000, [6]bool{false, false, false, true, true, true}},
		{"3999 (Resistance band)", 3999, [6]bool{false, false, false, true, true, true}},
		{"0", 0, [6]bool{false, false, false, false, false, false}},
		{"100", 100, [6]bool{false, false, false, false, false, false}},
		{"312", 312, [6]bool{false, false, false, false, false, false}},
		{"322", 322, [6]bool{false, false, false, false, false, false}},
		{"430 (Dual Blade)", 430, [6]bool{false, false, false, false, false, false}},
		{"434 (Dual Blade)", 434, [6]bool{false, false, false, false, false, false}},
		{"2000", 2000, [6]bool{false, false, false, false, false, false}},
		{"2100", 2100, [6]bool{false, false, false, false, false, false}},
		{"2300", 2300, [6]bool{false, false, false, false, false, false}},
	} {
		for i, v := range masterLevelVersions {
			want := c.want[i]
			t.Run(fmt.Sprintf("%s_v%d/%s", v.region, v.major, c.name), func(t *testing.T) {
				if got := UsesExtendedSP(c.jobId, v.region, v.major); got != want {
					t.Errorf("UsesExtendedSP(%d, %q, %d) = %v, want %v", c.jobId, v.region, v.major, got, want)
				}
			})
		}
	}
}

func TestUsesExtendedSP_LowerCaseRegion(t *testing.T) {
	if got := UsesExtendedSP(2218, "jms", 185); got != true {
		t.Errorf("UsesExtendedSP(2218, %q, 185) = %v, want true", "jms", got)
	}
}
