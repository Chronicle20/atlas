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
	"sync"
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

// zmapEntry holds the per-assetsRoot zmap and smap.
type zmapEntry struct {
	zmap []string
	smap map[string]string
}

// Compositor holds per-assetsRoot zmap/smap caches and a meta cache.
// zmaps and smaps are keyed by assetsRoot so multi-tenant processes never
// share zmap/smap data between tenants.
type Compositor struct {
	mu    sync.Mutex
	zmaps map[string]zmapEntry // assetsRoot -> {zmap, smap}
	cache *metaCache
}

// NewCompositor lazily loads zmap/smap from disk on first use.
func NewCompositor() *Compositor {
	return &Compositor{
		zmaps: make(map[string]zmapEntry),
		cache: newMetaCache(),
	}
}

func (c *Compositor) loadMaps(assetsRoot string) (zmapEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.zmaps[assetsRoot]; ok {
		return e, nil
	}

	z, err := LoadZmap(assetsRoot)
	if err != nil {
		return zmapEntry{}, err
	}
	s, err := LoadSmap(assetsRoot)
	if err != nil {
		return zmapEntry{}, err
	}
	e := zmapEntry{zmap: z, smap: s}
	c.zmaps[assetsRoot] = e
	return e, nil
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
	maps, err := c.loadMaps(req.AssetsRoot)
	if err != nil {
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
	if err := c.blitBody(canvas, req.AssetsRoot, bodyTemplate, stance, req.Frame, maps.zmap); err != nil {
		return nil, err
	}
	if err := c.blitEquipment(canvas, req, filtered, stance, maps.zmap); err != nil {
		return nil, err
	}

	out := NearestNeighborUpscale(canvas, req.Resize)
	return &CompositeResult{
		Image:             out,
		ResolvedStance:    stance,
		TwoHandedOverride: override,
		EquippedSlotCount: len(filtered),
	}, nil
}

// bodyTemplateId returns the on-disk directory name for a given gender + skin
// id. It mirrors normalizeId from the extractor: build the padded WZ name
// (0000{skin} or 0001{skin}) then strip leading zeros — so on-disk paths match
// what extractTemplateImg writes.
func bodyTemplateId(isMale bool, wzSkin int) string {
	prefix := "0000"
	if isMale {
		prefix = "0001"
	}
	full := fmt.Sprintf("%s%d", prefix, wzSkin)
	stripped := strings.TrimLeft(full, "0")
	if stripped == "" {
		return "0"
	}
	return stripped
}

// blitBody anchors the body's `body` part at the canvas center and draws
// every part canvas in the body img's frame in zmap order.
func (c *Compositor) blitBody(canvas *image.RGBA, assetsRoot, templateId, stance string, frame int, zmap []string) error {
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
		return zIndex(zmap, entries[i].meta.Z) < zIndex(zmap, entries[j].meta.Z)
	})
	for _, e := range entries {
		if err := drawPart(canvas, assetsRoot, templateId, stance, frame, e.part, e.anchor); err != nil {
			return err
		}
	}
	return nil
}

// zIndex returns the position of z in the zmap slice. Unknown values sort to
// the back so they appear on top of everything.
func zIndex(zmap []string, z string) int {
	for i, name := range zmap {
		if strings.EqualFold(name, z) {
			return i
		}
	}
	// Unknown z values sort to the back.
	return len(zmap)
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

// jointForSlot maps a render slot to the joint name on the body via which the
// equipment attaches. Slots not in this map are skipped.
var jointForSlot = map[int]string{
	-1:  "neck",  // hat — anchored via neck through head sprite chain (simplified: treat as neck)
	-2:  "neck",  // face
	-3:  "neck",  // eye accessory
	-4:  "neck",  // earrings
	-5:  "navel", // top
	-6:  "navel", // bottom
	-7:  "navel", // shoes (uses navel as a stand-in; v83 sprites use foot — refine in fixture if needed)
	-8:  "navel", // gloves (refined to "hand" in detail; navel is fallback)
	-9:  "navel", // cape
	-10: "hand",  // shield
	-11: "hand",  // weapon
	-12: "navel", // ring (no visual today — kept for completeness)
}

// blitEquipment iterates equipment in zmap order, resolves each part's joint
// anchor against the body, and blits the part canvases into `canvas`.
func (c *Compositor) blitEquipment(canvas *image.RGBA, req CompositeRequest, equipment map[int]int, stance string, zmap []string) error {
	bodyAnchor := Anchor{X: CanvasWidth / 2, Y: FootRow - 4}
	bodyTemplate := bodyTemplateId(req.IsMale, mustSkin(req.Skin))

	type entry struct {
		templateId string
		part       string
		meta       PartMeta
		anchor     Anchor
		zIdx       int
	}
	var entries []entry

	add := func(templateId string, jointFromBody string) error {
		parts, err := listFrameParts(req.AssetsRoot, templateId, stance, req.Frame)
		if err != nil {
			return err
		}
		for _, part := range parts {
			meta, ok := loadOrEmpty(req.AssetsRoot, templateId, stance, req.Frame, part)
			if !ok {
				continue
			}
			bodyJointMeta, _ := loadOrEmpty(req.AssetsRoot, bodyTemplate, stance, req.Frame, "body")
			originAnchor := ResolveAnchor(bodyAnchor, bodyJointMeta, meta, jointFromBody)
			// ResolveAnchor returns the canvas position of the child's origin.
			// drawPart places the sprite top-left at anchor, so subtract origin.
			anchor := Anchor{
				X: originAnchor.X - meta.Origin.X,
				Y: originAnchor.Y - meta.Origin.Y,
			}
			entries = append(entries, entry{
				templateId: templateId, part: part, meta: meta, anchor: anchor,
				zIdx: zIndex(zmap, meta.Z),
			})
		}
		return nil
	}

	// Hair / face anchored via neck.
	if req.Hair != 0 {
		if err := add(strconv.Itoa(req.Hair), "neck"); err != nil {
			return err
		}
	}
	if req.Face != 0 {
		if err := add(strconv.Itoa(req.Face), "neck"); err != nil {
			return err
		}
	}
	for slot, id := range equipment {
		joint, ok := jointForSlot[slot]
		if !ok {
			continue
		}
		if err := add(strconv.Itoa(id), joint); err != nil {
			return err
		}
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].zIdx < entries[j].zIdx })
	for _, e := range entries {
		if err := drawPart(canvas, req.AssetsRoot, e.templateId, stance, req.Frame, e.part, e.anchor); err != nil {
			return err
		}
	}
	return nil
}

// mustSkin returns the WZ id for a validated internal skin (caller ensures
// validity via MapInternalSkin upstream).
func mustSkin(internal int) int {
	id, _ := MapInternalSkin(internal)
	return id
}
