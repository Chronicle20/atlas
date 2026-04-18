package mapimage

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/property"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// DefaultMaxPixels is the default safety cap on canvas pixels (width * height).
const DefaultMaxPixels = 16384 * 16384

// Options configures a single Render call.
type Options struct {
	// MaxPixels caps width*height. Zero uses DefaultMaxPixels. Maps above the cap
	// return ErrSkipTooLarge.
	MaxPixels int
	// RenderBackgrounds toggles the back[] pre/post pass. Default true (matches PRD).
	RenderBackgrounds bool
}

// Stats is emitted for structured logging after a successful render.
type Stats struct {
	MapID       string
	Width       int
	Height      int
	SpriteCount int
	DurationMs  int64
	Output      string
}

// Render composites a map into {outDir}/map/{mapId}/render.png.
// Returns Stats, or ErrSkipEmpty / ErrSkipTooLarge wrapped errors for
// non-fatal skips so the caller can log and continue.
func Render(ctx context.Context, l logrus.FieldLogger, idx *Index, mapImg *wz.Image, outDir string, opts Options) (Stats, error) {
	if opts.MaxPixels == 0 {
		opts.MaxPixels = DefaultMaxPixels
	}
	mapId := normalizeMapId(mapImg.Name())
	start := time.Now()

	root := mapImg.Properties()
	info := childrenOf(root, "info")

	// Follow info/link — render the linked map under the alias id.
	if link := stringVal(info, "link", ""); link != "" {
		linked, ok := idx.maps[link]
		if !ok {
			padded := fmt.Sprintf("%09s", link)
			linked = idx.maps[padded]
		}
		if linked == nil {
			return Stats{MapID: mapId}, fmt.Errorf("map %s links to %s but target not found", mapId, link)
		}
		root = linked.Properties()
		info = childrenOf(root, "info")
	}

	if emptyMap(root) {
		return Stats{MapID: mapId}, ErrSkipEmpty
	}

	world, err := resolveBounds(info, root)
	if err != nil {
		return Stats{MapID: mapId}, fmt.Errorf("resolve bounds: %w", err)
	}
	if world.W <= 0 || world.H <= 0 {
		return Stats{MapID: mapId}, fmt.Errorf("invalid bounds %dx%d", world.W, world.H)
	}
	if world.W*world.H > opts.MaxPixels {
		return Stats{MapID: mapId, Width: world.W, Height: world.H},
			fmt.Errorf("%w: %dx%d exceeds MaxPixels=%d", ErrSkipTooLarge, world.W, world.H, opts.MaxPixels)
	}

	canvasImg := image.NewRGBA(image.Rect(0, 0, world.W, world.H))
	// Opaque black base; backgrounds paint over it.
	for i := 0; i < len(canvasImg.Pix); i += 4 {
		canvasImg.Pix[i+3] = 255
	}

	dec := newDecoder(idx.File)
	spriteCount := 0
	backs := loadBackEntries(root)

	if opts.RenderBackgrounds {
		spriteCount += renderBackgrounds(l, dec, idx, canvasImg, backs, world, 0)
	}

	for layer := 0; layer < 8; layer++ {
		if err := ctx.Err(); err != nil {
			return Stats{MapID: mapId}, err
		}
		layerSub := findSub(root, strconv.Itoa(layer))
		if layerSub == nil {
			continue
		}
		layerProps := layerSub.Children()
		layerInfo := childrenOf(layerProps, "info")
		tS := stringVal(layerInfo, "tS", "")

		// Objs first (background deco)…
		objs := loadObjEntries(layerProps)
		orecs := make([]objRec, 0, len(objs))
		for _, o := range objs {
			s, _, err := idx.resolveObjSprite(dec, o.oS, o.l0, o.l1, o.l2)
			if err != nil {
				l.Debugf("obj layer=%d idx=%d (%s/%s/%s/%s): %v", layer, o.idx, o.oS, o.l0, o.l1, o.l2, err)
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
			blit(canvasImg, src, or.e.x, or.e.y, or.s.ox, or.s.oy, world, or.s.w, or.s.h)
			spriteCount++
		}

		// …then tiles on top.
		tiles := loadTileEntries(layerProps, tS)
		trecs := make([]tileRec, 0, len(tiles))
		for _, t := range tiles {
			if t.tS == "" {
				continue
			}
			s, _, err := idx.resolveTileSprite(dec, t.tS, t.u, t.no)
			if err != nil {
				l.Debugf("tile layer=%d idx=%d (%s/%s/%s): %v", layer, t.idx, t.tS, t.u, t.no, err)
				continue
			}
			trecs = append(trecs, tileRec{e: t, s: s})
		}
		sortTileRecs(trecs)
		for _, tr := range trecs {
			blit(canvasImg, tr.s.img, tr.e.x, tr.e.y, tr.s.ox, tr.s.oy, world, tr.s.w, tr.s.h)
			spriteCount++
		}
	}

	if opts.RenderBackgrounds {
		spriteCount += renderBackgrounds(l, dec, idx, canvasImg, backs, world, 1)
	}

	dir := filepath.Join(outDir, "map", mapId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Stats{MapID: mapId}, fmt.Errorf("mkdir: %w", err)
	}
	outPath := filepath.Join(dir, "render.png")
	out, err := os.Create(outPath)
	if err != nil {
		return Stats{MapID: mapId}, fmt.Errorf("create: %w", err)
	}
	defer out.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(out, canvasImg); err != nil {
		return Stats{MapID: mapId}, fmt.Errorf("encode PNG: %w", err)
	}

	return Stats{
		MapID:       mapId,
		Width:       world.W,
		Height:      world.H,
		SpriteCount: spriteCount,
		DurationMs:  time.Since(start).Milliseconds(),
		Output:      outPath,
	}, nil
}

// renderBackgrounds runs either the front=0 or front=1 pass. Returns the
// count of sprites successfully composited.
func renderBackgrounds(l logrus.FieldLogger, dec *decoder, idx *Index, canvasImg *image.RGBA, backs []backEntry, world WorldBounds, front int) int {
	count := 0
	for _, b := range backs {
		if b.front != front {
			continue
		}
		s, cp, err := idx.resolveBackSprite(dec, b.bS, b.no)
		if err != nil {
			l.Debugf("back idx=%d (%s/%d): %v", b.idx, b.bS, b.no, err)
			continue
		}
		src := s.img
		if b.f != 0 {
			src = dec.mirrored(cp, s.img)
		}
		drawBackground(canvasImg, src, s, b, world)
		count++
	}
	return count
}

// emptyMap reports whether a map has no back[] and no layer content worth rendering.
func emptyMap(root []property.Property) bool {
	if back := findSub(root, "back"); back != nil && len(back.Children()) > 0 {
		return false
	}
	for layer := 0; layer < 8; layer++ {
		ls := findSub(root, strconv.Itoa(layer))
		if ls == nil {
			continue
		}
		if obj := findSub(ls.Children(), "obj"); obj != nil && len(obj.Children()) > 0 {
			return false
		}
		if tile := findSub(ls.Children(), "tile"); tile != nil && len(tile.Children()) > 0 {
			return false
		}
	}
	return true
}

// normalizeMapId strips leading zeros to produce a numeric string.
func normalizeMapId(id string) string {
	out := id
	for len(out) > 1 && out[0] == '0' {
		out = out[1:]
	}
	return out
}
