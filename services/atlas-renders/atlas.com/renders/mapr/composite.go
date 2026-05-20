package mapr

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"atlas-renders/storage"

	"github.com/sirupsen/logrus"
)

// Composite stacks pre-composited per-layer PNGs in layout.ZMap order onto a
// world-sized canvas. The layer PNGs are produced by atlas-data ingest
// (libs/atlas-wz/mapimage.ExtractLayers) and are already world-anchored —
// i.e. sized to the map bounds with sprites placed at their world positions.
// So the render-time job here is purely a stacked draw.Over.
//
// STATED LIMITATION: this composite intentionally omits the Map.img back[]
// (background tile/skybox) imagery. Backgrounds are not currently persisted
// alongside layout.json by ingest; when that pipeline is extended the
// background can be blitted before the foreground zmap stack here. The
// resulting render today is the foreground world only on a transparent
// canvas.
func Composite(l logrus.FieldLogger, m *storage.MapEntry) (image.Image, error) {
	if m == nil {
		return nil, fmt.Errorf("composite: nil map entry")
	}
	bounds := image.Rect(m.Layout.Bounds.Left, m.Layout.Bounds.Top, m.Layout.Bounds.Right, m.Layout.Bounds.Bottom)
	if bounds.Empty() {
		return nil, fmt.Errorf("map %d: empty bounds", m.Layout.MapID)
	}

	canvas := image.NewNRGBA(bounds)

	// Index layers by name for zmap lookup.
	layerByName := make(map[string]int, len(m.Layout.Layers))
	for _, layer := range m.Layout.Layers {
		layerByName[layer.Name] = layer.ID
	}

	// Walk zmap back-to-front and blit each referenced layer PNG.
	order := m.Layout.ZMap
	if len(order) == 0 {
		// No explicit zmap — fall back to layer declaration order.
		order = make([]string, 0, len(m.Layout.Layers))
		for _, layer := range m.Layout.Layers {
			order = append(order, layer.Name)
		}
	}

	for _, name := range order {
		layerID, ok := layerByName[name]
		if !ok {
			l.Warnf("map %d: zmap references unknown layer %q", m.Layout.MapID, name)
			continue
		}
		pngBytes, ok := m.Layers[layerID]
		if !ok {
			continue
		}
		img, err := png.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			l.WithError(err).Warnf("map %d layer %d: decode failed", m.Layout.MapID, layerID)
			continue
		}
		// Layer PNGs are world-sized; place at the canvas origin.
		draw.Draw(canvas, bounds, img, img.Bounds().Min, draw.Over)
	}

	return canvas, nil
}
