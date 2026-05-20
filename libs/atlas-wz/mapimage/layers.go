package mapimage

import (
	"fmt"
	"image"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// LayerOutput is one composited layer PNG plus its z-position. The library
// composites tiles + objects within a single layer (using internal blit/sort/
// bounds helpers); atlas-renders later stacks LayerOutput images in zmap
// order at request time.
type LayerOutput struct {
	ID    int
	Z     int
	Image image.Image
	Name  string // bucket-key suffix, e.g. "layer-0"
}

// maxLayers caps the number of numbered layer subtrees (0..maxLayers-1).
// MapleStory map images use eight layers; matches donor renderer.go.
const maxLayers = 8

// ExtractLayers walks a parsed Map.img and returns one composited image per
// numbered layer (0..7) that has at least one tile or obj, plus a
// maplayout.Layout containing bounds/footholds/portals/NPCs/zmap.
//
// Each LayerOutput.Image is sized to the resolved world bounds; sprites are
// blitted at their world-anchored positions using draw.Over so transparent
// pixels accumulate cleanly when the consumer later stacks layers. No
// background (back[]) is drawn; backgrounds are stacked by atlas-renders at
// render time.
//
// idx may be nil; when nil, sprite resolution falls back to an Index built
// from img.File(), which only works when the Map.wz file has been parsed in
// full (including the Back/Tile/Obj sub-directories).
func ExtractLayers(idx *Index, img *wz.Image) ([]LayerOutput, maplayout.Layout, error) {
	if img == nil {
		return nil, maplayout.Layout{}, fmt.Errorf("layers: nil image")
	}
	root := img.Properties()
	info := childrenOf(root, "info")

	bounds, err := resolveBounds(info, root)
	if err != nil {
		return nil, maplayout.Layout{}, fmt.Errorf("resolve bounds: %w", err)
	}
	if bounds.W <= 0 || bounds.H <= 0 {
		return nil, maplayout.Layout{}, fmt.Errorf("invalid bounds %dx%d", bounds.W, bounds.H)
	}

	layout := maplayout.Layout{
		Version:   maplayout.SchemaVersion,
		MapID:     parseMapID(img.Name()),
		Bounds:    maplayout.Bounds{Left: bounds.X, Top: bounds.Y, Right: bounds.X + bounds.W, Bottom: bounds.Y + bounds.H},
		Footholds: extractFootholds(root),
		Portals:   extractPortals(root),
		NPCs:      extractNPCs(root),
	}

	if idx == nil && img.File() != nil {
		idx = NewIndex(img.File())
	}

	outputs := make([]LayerOutput, 0, maxLayers)
	layerMetas := make([]maplayout.Layer, 0, maxLayers)
	for layer := 0; layer < maxLayers; layer++ {
		layerSub := findSub(root, strconv.Itoa(layer))
		if layerSub == nil {
			continue
		}
		layerProps := layerSub.Children()
		objs := loadObjEntries(layerProps)
		layerInfo := childrenOf(layerProps, "info")
		tS := stringVal(layerInfo, "tS", "")
		tiles := loadTileEntries(layerProps, tS)
		if len(objs) == 0 && len(tiles) == 0 {
			continue
		}

		layerImg, err := compositeLayer(idx, bounds, layerProps, objs, tiles)
		if err != nil {
			return nil, maplayout.Layout{}, fmt.Errorf("layer %d: %w", layer, err)
		}

		name := fmt.Sprintf("layer-%d", layer)
		outputs = append(outputs, LayerOutput{
			ID:    layer,
			Z:     layer,
			Image: layerImg,
			Name:  name,
		})
		layerMetas = append(layerMetas, maplayout.Layer{
			ID:     layer,
			Name:   name,
			Z:      layer,
			Source: name,
		})
	}

	layout.Layers = layerMetas
	return outputs, layout, nil
}

// compositeLayer renders one layer's tile + obj entries into a world-sized
// RGBA image. Objects are drawn first (background deco), then tiles on top,
// matching the donor renderer's intra-layer order.
func compositeLayer(idx *Index, world WorldBounds, layerProps []property.Property, objs []objEntry, tiles []tileEntry) (image.Image, error) {
	out := image.NewRGBA(image.Rect(0, 0, world.W, world.H))

	if idx == nil {
		// No backing index — return an empty transparent canvas. The Layout
		// is still useful; atlas-renders can decide what to do with empty
		// layer images.
		return out, nil
	}

	dec := newDecoder(idx.File)

	// Objects first (decoration behind tiles).
	orecs := make([]objRec, 0, len(objs))
	for _, o := range objs {
		s, _, err := idx.resolveObjSprite(dec, o.oS, o.l0, o.l1, o.l2)
		if err != nil {
			continue
		}
		orecs = append(orecs, objRec{e: o, s: s})
	}
	sortObjRecs(orecs)
	for _, or := range orecs {
		src := or.s.img
		if or.e.f != 0 {
			src = mirrorX(or.s.img)
		}
		blit(out, src, or.e.x, or.e.y, or.s.ox, or.s.oy, world, or.s.w, or.s.h)
	}

	// Tiles on top.
	trecs := make([]tileRec, 0, len(tiles))
	for _, t := range tiles {
		if t.tS == "" {
			continue
		}
		s, _, err := idx.resolveTileSprite(dec, t.tS, t.u, t.no)
		if err != nil {
			continue
		}
		trecs = append(trecs, tileRec{e: t, s: s})
	}
	sortTileRecs(trecs)
	for _, tr := range trecs {
		blit(out, tr.s.img, tr.e.x, tr.e.y, tr.s.ox, tr.s.oy, world, tr.s.w, tr.s.h)
	}

	return out, nil
}

// extractFootholds walks foothold/{group}/{poly}/{seg} into a flat list.
// Each segment becomes one Foothold; Prev/Next aren't encoded in WZ shape
// directly but live alongside the segment id, so we read them as
// "prev"/"next" int values.
func extractFootholds(root []property.Property) []maplayout.Foothold {
	fh := findSub(root, "foothold")
	if fh == nil {
		return nil
	}
	var out []maplayout.Foothold
	for _, group := range fh.Children() {
		gsub, ok := group.(*property.SubProperty)
		if !ok {
			continue
		}
		for _, poly := range gsub.Children() {
			psub, ok := poly.(*property.SubProperty)
			if !ok {
				continue
			}
			for _, seg := range psub.Children() {
				ssub, ok := seg.(*property.SubProperty)
				if !ok {
					continue
				}
				ch := ssub.Children()
				id, _ := strconv.Atoi(ssub.Name())
				out = append(out, maplayout.Foothold{
					ID:   id,
					X1:   intVal(ch, "x1", 0),
					Y1:   intVal(ch, "y1", 0),
					X2:   intVal(ch, "x2", 0),
					Y2:   intVal(ch, "y2", 0),
					Prev: intVal(ch, "prev", 0),
					Next: intVal(ch, "next", 0),
				})
			}
		}
	}
	return out
}

// extractPortals walks portal/<i>/ entries into a flat list.
func extractPortals(root []property.Property) []maplayout.Portal {
	portal := findSub(root, "portal")
	if portal == nil {
		return nil
	}
	var out []maplayout.Portal
	for _, p := range portal.Children() {
		sub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		target := uint32(intVal(ch, "tm", 0))
		out = append(out, maplayout.Portal{
			Name:   stringVal(ch, "pn", ""),
			Type:   intVal(ch, "pt", 0),
			Target: target,
			X:      intVal(ch, "x", 0),
			Y:      intVal(ch, "y", 0),
		})
	}
	return out
}

// extractNPCs walks life/<i>/ entries whose type == "n" into NPC records.
func extractNPCs(root []property.Property) []maplayout.NPC {
	life := findSub(root, "life")
	if life == nil {
		return nil
	}
	var out []maplayout.NPC
	for _, p := range life.Children() {
		sub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		if stringVal(ch, "type", "") != "n" {
			continue
		}
		id, _ := strconv.Atoi(stringVal(ch, "id", "0"))
		// id may be stored as an int too; fall back.
		if id == 0 {
			id = intVal(ch, "id", 0)
		}
		out = append(out, maplayout.NPC{
			ID:       uint32(id),
			X:        intVal(ch, "x", 0),
			Y:        intVal(ch, "cy", 0),
			Foothold: intVal(ch, "fh", 0),
		})
	}
	return out
}

// parseMapID parses a Map.img name like "100000000" or "000100000000" into
// a numeric id. Returns 0 on failure.
func parseMapID(name string) uint32 {
	// Strip leading zeros, but keep "0" as 0.
	trimmed := name
	for len(trimmed) > 1 && trimmed[0] == '0' {
		trimmed = trimmed[1:]
	}
	v, err := strconv.ParseUint(trimmed, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(v)
}
