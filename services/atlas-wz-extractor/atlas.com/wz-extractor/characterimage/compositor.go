package characterimage

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CanvasWidth and CanvasHeight are the native, pre-resize compositing canvas
// dimensions. The body origin lands at (CanvasWidth/2, FootRow - 4).
const (
	CanvasWidth  = 96
	CanvasHeight = 128
	FootRow      = 124
)

// CompositeRequest describes one render. All slot filtering and skin mapping
// is done by the compositor — callers pass the raw request shape from the
// HTTP layer.
type CompositeRequest struct {
	AssetsRoot string      // absolute path: {OUTPUT_IMG_DIR}/{tenant}/{region}/{v}
	Skin       int         // internal 0..10
	Hair       int         // hair templateId
	Face       int         // face templateId
	Equipment  map[int]int // raw slot -> templateId
	Stance     string      // requested stance; may be overridden
	Frame      int
	Resize     int  // 1..4
	IsMale     bool // if true, use 0001{wzSkin}.img; else 0000{wzSkin}.img
}

// Compositor holds the per-process zmap/smap and meta cache.
type Compositor struct {
	zmap  []string
	smap  map[string]string
	cache *metaCache
}

// NewCompositor lazily loads zmap/smap from disk on first use.
func NewCompositor() *Compositor {
	return &Compositor{cache: newMetaCache()}
}

func (c *Compositor) loadMaps(assetsRoot string) error {
	if c.zmap == nil {
		z, err := LoadZmap(assetsRoot)
		if err != nil {
			return err
		}
		c.zmap = z
	}
	if c.smap == nil {
		s, err := LoadSmap(assetsRoot)
		if err != nil {
			return err
		}
		c.smap = s
	}
	return nil
}

// CompositeResult bundles the composited image and observability metadata.
type CompositeResult struct {
	Image             *image.RGBA
	ResolvedStance    string
	TwoHandedOverride bool
	EquippedSlotCount int
}

// Composite runs the algorithm:
//  1. filter equipment, 2. resolve stance, 3. map skin, 4. validate stance/frame,
//  5. blit body skin, 6. blit equipment by zmap order, 7. scale.
func (c *Compositor) Composite(req CompositeRequest) (*CompositeResult, error) {
	if err := ValidateStance(req.Stance); err != nil {
		return nil, err
	}
	if req.Resize < 1 || req.Resize > 4 {
		return nil, fmt.Errorf("resize out of range 1..4: %d", req.Resize)
	}
	if err := c.loadMaps(req.AssetsRoot); err != nil {
		return nil, err
	}

	filtered := FilterEquipment(req.Equipment)
	stance, override := ResolveStance(req.Stance, filtered)

	wzSkin, err := MapInternalSkin(req.Skin)
	if err != nil {
		return nil, err
	}
	bodyTemplate := bodyTemplateId(req.IsMale, wzSkin)

	if err := ValidateFrame(req.AssetsRoot, bodyTemplate, stance, req.Frame); err != nil {
		return nil, err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))
	if err := c.blitBody(canvas, req.AssetsRoot, bodyTemplate, stance, req.Frame); err != nil {
		return nil, err
	}
	// Equipment blitting comes in Task 5.9.

	out := NearestNeighborUpscale(canvas, req.Resize)
	return &CompositeResult{
		Image:             out,
		ResolvedStance:    stance,
		TwoHandedOverride: override,
		EquippedSlotCount: len(filtered),
	}, nil
}

// bodyTemplateId returns the WZ img name for a given gender + skin id.
// Female: 0000{skin}, male: 0001{skin}.
func bodyTemplateId(isMale bool, wzSkin int) string {
	prefix := "0000"
	if isMale {
		prefix = "0001"
	}
	return fmt.Sprintf("%s%d", prefix, wzSkin)
}

// blitBody anchors the body's `body` part at the canvas center and draws
// every part canvas in the body img's frame in zmap order.
func (c *Compositor) blitBody(canvas *image.RGBA, assetsRoot, templateId, stance string, frame int) error {
	bodyAnchor := Anchor{X: CanvasWidth / 2, Y: FootRow - 4}

	parts, err := listFrameParts(assetsRoot, templateId, stance, frame)
	if err != nil {
		return err
	}
	bodyMeta, hasBody := loadOrEmpty(assetsRoot, templateId, stance, frame, "body")
	if !hasBody {
		// Some sprites use "neck" or other names; fall back to first part.
		if len(parts) == 0 {
			return fmt.Errorf("%w: body sprite has no parts", ErrAssetsMissing)
		}
		bodyMeta, _ = loadOrEmpty(assetsRoot, templateId, stance, frame, parts[0])
	}

	type entry struct {
		part   string
		meta   PartMeta
		anchor Anchor
	}
	var entries []entry
	for _, part := range parts {
		meta, _ := loadOrEmpty(assetsRoot, templateId, stance, frame, part)
		anchor := Anchor{
			X: bodyAnchor.X - meta.Origin.X,
			Y: bodyAnchor.Y - meta.Origin.Y,
		}
		// All body parts share the body's origin frame — skip joint walk.
		_ = bodyMeta
		entries = append(entries, entry{part, meta, anchor})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return c.zIndex(entries[i].meta.Z) < c.zIndex(entries[j].meta.Z)
	})
	for _, e := range entries {
		if err := drawPart(canvas, assetsRoot, templateId, stance, frame, e.part, e.anchor); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compositor) zIndex(z string) int {
	for i, name := range c.zmap {
		if strings.EqualFold(name, z) {
			return i
		}
	}
	// Unknown z values sort to the back.
	return len(c.zmap)
}

func loadOrEmpty(assetsRoot, templateId, stance string, frame int, part string) (PartMeta, bool) {
	pm, err := LoadPartMeta(assetsRoot, templateId, stance, frame, part)
	if err != nil {
		return PartMeta{}, false
	}
	return pm, true
}

func listFrameParts(assetsRoot, templateId, stance string, frame int) ([]string, error) {
	dir := filepath.Join(assetsRoot, "character-parts", templateId, stance, strconv.Itoa(frame))
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrAssetsMissing, dir)
		}
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}
	var out []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasSuffix(name, ".png") {
			out = append(out, strings.TrimSuffix(name, ".png"))
		}
	}
	return out, nil
}

func drawPart(canvas *image.RGBA, assetsRoot, templateId, stance string, frame int, part string, anchor Anchor) error {
	path := filepath.Join(assetsRoot, "character-parts", templateId, stance, strconv.Itoa(frame), part+".png")
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open part %s: %w", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("decode part %s: %w", path, err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dr := image.Rect(anchor.X, anchor.Y, anchor.X+w, anchor.Y+h)
	draw.Draw(canvas, dr, img, image.Point{}, draw.Over)
	return nil
}
