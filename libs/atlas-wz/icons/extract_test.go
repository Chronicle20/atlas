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
