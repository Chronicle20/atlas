package mapimage

import (
	"fmt"
	"image"

	"github.com/Chronicle20/atlas/libs/atlas-wz/canvas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// ExtractMinimap returns the decoded miniMap/canvas of a parsed Map.img.
// Returns ErrNoMinimap when no miniMap canvas property exists.
func ExtractMinimap(img *wz.Image) (image.Image, error) {
	if img == nil {
		return nil, ErrNoMinimap
	}
	props, err := img.Properties()
	if err != nil {
		return nil, fmt.Errorf("minimap properties: %w", err)
	}
	cp := findMinimapCanvas(props)
	if cp == nil {
		return nil, ErrNoMinimap
	}
	f := img.File()
	if f == nil {
		return nil, fmt.Errorf("minimap: image has no backing wz.File")
	}
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return nil, fmt.Errorf("read canvas data: %w", err)
	}
	out, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("decompress canvas: %w", err)
	}
	return out, nil
}

// findMinimapCanvas returns the miniMap/canvas property, or nil if absent.
func findMinimapCanvas(props []property.Property) *property.CanvasProperty {
	mm := findSub(props, "miniMap")
	if mm == nil {
		return nil
	}
	return findCanvas(mm.Children(), "canvas")
}

// extractZmap parses a zmap.img property tree into an ordered slice of
// layer-string names. Order in the WZ is the render order.
//
// best-effort: returns nil when img.Properties() fails to parse so callers
// fall back to layer-declaration order instead of aborting layout extraction.
func extractZmap(img *wz.Image) []string {
	if img == nil {
		return nil
	}
	props, err := img.Properties()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(props))
	for _, p := range props {
		out = append(out, p.Name())
	}
	return out
}
