package mapr

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/Chronicle20/atlas/libs/atlas-wz/mapimage"
	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
)

// CompositeFromWZ composites a map render directly from a parsed *wz.File
// holding Map.wz. The flow is:
//
//  1. Build a per-Map.wz Index for Back/Tile/Obj/Map lookup.
//  2. Locate the requested mapId's .img in idx.Maps().
//  3. Call mapimage.ExtractLayers to lazily parse + composite each numbered
//     layer (0..7) into a world-sized RGBA image. Sprite resolution lazily
//     walks the Tile/Obj sub-trees.
//  4. Stack the produced layer images in layout.ZMap order with draw.Over
//     onto a canvas sized to the layout bounds. When layout.ZMap is empty
//     (atlas-data ingest does not populate it; the only stable order
//     available is the layer declaration), fall back to layer declaration
//     order from layout.Layers.
//
// STATED LIMITATION (carried from the pre-refactor composite): backgrounds
// (Map.img back[]) are not blitted. The composite produced here is the
// foreground world on a transparent canvas, same as before.
//
// Returns an error if the map's .img is not found, if ExtractLayers fails
// (most often "resolve bounds: no bounds" for stub/test maps), or if the
// resulting canvas has empty bounds.
func CompositeFromWZ(l logrus.FieldLogger, file *wz.File, layout maplayout.Layout, mapID uint32) (image.Image, error) {
	if file == nil {
		return nil, fmt.Errorf("composite: nil wz file")
	}

	idx := mapimage.NewIndex(file)
	mapImg, ok := idx.Maps()[fmt.Sprintf("%09d", mapID)]
	if !ok {
		// Fallback: try the non-padded form. WZ image names in v83 are
		// zero-padded to 9 digits ("00100000000" → "100000000" after strip),
		// but be defensive in case future revisions diverge.
		for name, img := range idx.Maps() {
			if name == fmt.Sprintf("%d", mapID) {
				mapImg = img
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("map %d: image not found in Map.wz", mapID)
	}

	layers, _, err := mapimage.ExtractLayers(idx, mapImg)
	if err != nil {
		return nil, fmt.Errorf("map %d extract layers: %w", mapID, err)
	}

	bounds := image.Rect(layout.Bounds.Left, layout.Bounds.Top, layout.Bounds.Right, layout.Bounds.Bottom)
	if bounds.Empty() {
		return nil, fmt.Errorf("map %d: empty bounds", mapID)
	}
	canvas := image.NewNRGBA(bounds)

	// Build a name → LayerOutput lookup so the zmap-order traversal can
	// fetch each layer's image by name. layout.Layers + layers may not be
	// in 1:1 order if ExtractLayers and ExtractLayout disagree on which
	// layers have content; trust ExtractLayers' output for sourcing.
	byName := make(map[string]image.Image, len(layers))
	for _, lo := range layers {
		byName[lo.Name] = lo.Image
	}

	order := layout.ZMap
	if len(order) == 0 {
		order = make([]string, 0, len(layout.Layers))
		for _, layer := range layout.Layers {
			order = append(order, layer.Name)
		}
	}
	for _, name := range order {
		img, ok := byName[name]
		if !ok {
			// Layout listed this layer but ExtractLayers didn't produce it;
			// happens when a layer has zero tiles+objs. Skip silently.
			continue
		}
		draw.Draw(canvas, bounds, img, img.Bounds().Min, draw.Over)
	}

	if l != nil {
		l.Debugf("composite: map %d stacked %d layer(s)", mapID, len(byName))
	}
	return canvas, nil
}
