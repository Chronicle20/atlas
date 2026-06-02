package charparts

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// ErrZmapMissing is returned by ExtractZmap when the supplied Base.wz file does
// not contain a "zmap.img" entry at its root. Callers may log and continue; the
// absence of zmap.json downstream makes atlas-renders fall back to insertion
// order for character part z-ordering (weapons/shields/accessories layer
// arbitrarily — the pre-fix behaviour).
var ErrZmapMissing = errors.New("charparts: zmap.img not found")

// ExtractZmap reads Base.wz/zmap.img and returns the ordered list of layer
// names. The order in the WZ IS the render order: it is front-to-back, so
// index 0 is the most-frontward layer and the last entry is the most-backward.
// atlas-renders resolves each sprite's render-layer label (manifest.Sprite.Z,
// the canvas's `z` child — the same vocabulary smap.img is keyed on) to its
// index here and draws back-most first. Note this is the `z` label, NOT the
// canvas name (manifest.Sprite.Part), which is often generic (e.g. "weapon").
//
// Returns ErrZmapMissing if zmap.img cannot be located in the file's root.
// Donor: services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap.go
// writeZmap / writeZmapFromProps.
func ExtractZmap(base *wz.File) ([]string, error) {
	if base == nil {
		return nil, fmt.Errorf("charparts.ExtractZmap: nil wz.File")
	}
	root := base.Root()
	if root == nil {
		return nil, ErrZmapMissing
	}
	zmapImg := findZmapImage(root.Images())
	if zmapImg == nil {
		return nil, ErrZmapMissing
	}
	props, err := zmapImg.Properties()
	if err != nil {
		return nil, fmt.Errorf("zmap properties: %w", err)
	}
	return zmapFromProps(props), nil
}

// zmapFromProps is the pure transformation: each child of zmap.img contributes
// its name to the ordered slice, preserving WZ declaration order (= render
// order). Donor: writeZmapFromProps.
func zmapFromProps(props []property.Property) []string {
	out := make([]string, 0, len(props))
	for _, p := range props {
		out = append(out, p.Name())
	}
	return out
}

// findZmapImage returns the root-level image named "zmap" (case-insensitive,
// ".img"-suffix tolerant), or nil. Shares eqFoldStripImg with smap.go.
func findZmapImage(images []*wz.Image) *wz.Image {
	for _, img := range images {
		if eqFoldStripImg(img.Name(), "zmap") {
			return img
		}
	}
	return nil
}

// MarshalZmap serializes the ordered zmap slice. A nil slice marshals to "[]"
// so the downstream PUT never writes "null" bytes. Order is preserved verbatim
// — it is semantically significant (render order).
func MarshalZmap(z []string) ([]byte, error) {
	if z == nil {
		z = []string{}
	}
	return json.Marshal(z)
}
