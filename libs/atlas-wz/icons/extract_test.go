package icons

import "testing"

// TestPublicSurfaceExists is a compile-time check on the public API.
// Richer fixture-based tests live in atlas-wz-extractor where WZ fixtures
// are wired up; the lib itself ships without binary fixtures.
func TestPublicSurfaceExists(t *testing.T) {
	_ = ExtractItemIcon
	_ = ExtractNpcIcon
	_ = ExtractMobIcon
	_ = ExtractReactorIcon
	_ = ExtractSkillIcon
	_ = ErrNotFound
}

// TestNormalizeId locks in the behavior that extractEntityIcon depends on:
// the image-loop comparison `normalizeId(img.Name()) == target` must match
// regardless of the ".img" suffix WZ image names carry, and regardless of
// zero-padding the WZ uses for ids below the canonical width.
func TestNormalizeId(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Real Mob.wz / Npc.wz / Reactor.wz image names — these had been
		// silently failing every icon extract on PR-544 because the
		// trailing ".img" prevented the equality comparison from matching
		// the numeric target.
		{"0100100.img", "100100"},
		{"00009300.img", "9300"},
		{"9300100.img", "9300100"},
		// UOL info/link values — typically the raw id, sometimes padded.
		{"100100", "100100"},
		{"0100100", "100100"},
		// Edge: all zeros (a real id 0 is unusual but legal).
		{"0000000.img", "0"},
		{"0", "0"},
	}
	for _, tc := range cases {
		got := normalizeId(tc.in)
		if got != tc.want {
			t.Errorf("normalizeId(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
