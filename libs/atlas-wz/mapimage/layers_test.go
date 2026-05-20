package mapimage

import "testing"

// TestPublicSurfaceExists is a compile-time check on the public API.
// Richer tests live in atlas-wz-extractor where WZ fixtures are wired up.
func TestPublicSurfaceExists(t *testing.T) {
	_ = ExtractLayers
	_ = ExtractMinimap
	_ = LayerOutput{}
	_ = ErrNoMinimap
	_ = ErrSkipEmpty
	_ = ErrSkipTooLarge
}
