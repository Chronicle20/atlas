package mapimage

import "testing"

// TestPublicSurfaceExists is a compile-time check on the public API.
// Richer tests live in atlas-wz-extractor where WZ fixtures are wired up.
func TestPublicSurfaceExists(t *testing.T) {
	_ = ExtractLayers
	_ = ExtractLayout
	_ = ExtractMinimap
	_ = LayerOutput{}
	_ = ErrNoMinimap
	_ = ErrSkipEmpty
	_ = ErrSkipTooLarge
}

// TestExtractLayout_NilImage proves the metadata-only entry point refuses a
// nil input rather than panicking — it's called per-map in the ingest loop
// and a panic would tear the worker down.
func TestExtractLayout_NilImage(t *testing.T) {
	if _, err := ExtractLayout(nil); err == nil {
		t.Fatal("ExtractLayout(nil) expected error, got nil")
	}
}
