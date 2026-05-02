package characterimage

import (
	"errors"
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
// dimensions. The body's `body` part origin lands at (CanvasWidth/2, FootRow-4).
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
//  5. seed body skin, 6. add head + equipment + hair/face via shared joints,
//  7. zmap-sort + draw.
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
	if err := c.placeAndDraw(canvas, maps.zmap, req, bodyTemplate, stance); err != nil {
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

// headTemplateId returns the on-disk directory name for the head template
// associated with a given WZ skin id. Heads are named 0001{wzSkin}.img in
// Character.wz (note: this is also the male body skin pattern — but heads
// store their canvas under front/head and back/head, never under stand1).
// After normalizeId strips leading zeros, the on-disk dir matches.
func headTemplateId(wzSkin int) string {
	full := fmt.Sprintf("0001%d", wzSkin)
	stripped := strings.TrimLeft(full, "0")
	if stripped == "" {
		return "0"
	}
	return stripped
}

// placement is a positioned, ready-to-draw part: source coords plus the
// metadata needed for joint chaining and z-ordering.
type placement struct {
	templateId string
	stance     string
	frame      int
	partName   string
	meta       PartMeta
	// anchor is the canvas position where this part's `origin` lands.
	// drawPart blits sprite top-left at (anchor - origin).
	anchor Anchor
}

// placeAndDraw seeds the joint graph with the body skin, attaches the head
// template, every equipped slot, plus optional hair and face, then renders
// every placement in zmap z-order.
func (c *Compositor) placeAndDraw(canvas *image.RGBA, zmap []string, req CompositeRequest, bodyTemplate, stance string) error {
	wzSkin := mustSkin(req.Skin)
	filtered := FilterEquipment(req.Equipment)
	placed := make([]placement, 0, 32)
	owners := make([]vslotOwner, 0, 8)

	// 1. Body skin parts under the resolved stance.
	if err := c.appendBodyParts(&placed, req.AssetsRoot, bodyTemplate, stance, req.Frame); err != nil {
		return err
	}
	owners = appendOwner(owners, c, req.AssetsRoot, bodyTemplate, ownerBody)

	// 2. Head template — head canvas always lives under front/0 (we extract
	//    front/head as a direct-canvas stance into front/0/head.{png,json}).
	headTmpl := headTemplateId(wzSkin)
	if err := c.appendTemplateParts(&placed, req.AssetsRoot, headTmpl, "front", 0, true); err != nil {
		return err
	}
	owners = appendOwner(owners, c, req.AssetsRoot, headTmpl, ownerHead)

	// 3. Equipment in iteration order. Slot order isn't meaningful here —
	//    z-ordering happens at draw time. The graph chain naturally lets each
	//    new piece find the most-recent compatible parent.
	for _, id := range filtered {
		if id == 0 {
			continue
		}
		if err := c.appendEquipmentParts(&placed, req, id, stance); err != nil {
			return err
		}
		owners = appendOwner(owners, c, req.AssetsRoot, strconv.Itoa(id), ownerEquipment)
	}

	// 4. Hair + face are sourced like equipment (their assets typically live
	//    under default/0). They attach to the head via shared joints (brow,
	//    earOverHead, etc.) — no special-casing required.
	if req.Hair != 0 {
		if err := c.appendEquipmentParts(&placed, req, req.Hair, stance); err != nil {
			return err
		}
		owners = appendOwner(owners, c, req.AssetsRoot, strconv.Itoa(req.Hair), ownerHair)
	}
	if req.Face != 0 {
		if err := c.appendEquipmentParts(&placed, req, req.Face, stance); err != nil {
			return err
		}
		owners = appendOwner(owners, c, req.AssetsRoot, strconv.Itoa(req.Face), ownerFace)
	}

	// 5. Apply vslot/smap occlusion so equipment claims (e.g. a full helmet's
	//    "CpH1H2H3H4H5HfHsHbAe") suppress hair parts in the slots it covers.
	smap, _ := c.loadMaps(req.AssetsRoot) // already loaded above; ignore err
	sort.SliceStable(owners, func(i, j int) bool { return owners[i].kind < owners[j].kind })
	placed = applyVslotOcclusion(placed, smap.smap, owners)

	// 6. Sort by zmap order and blit. The zmap is ordered front-to-back —
	// lower index = more frontward. Drawing iterates placed in order, so
	// back-most must come first. Sort descending: highest zIndex (back-most)
	// drawn first, lowest zIndex (front-most) drawn last.
	sort.SliceStable(placed, func(i, j int) bool {
		return zIndex(zmap, placed[i].meta.Z) > zIndex(zmap, placed[j].meta.Z)
	})
	for _, p := range placed {
		blitTopLeft := Anchor{
			X: p.anchor.X - p.meta.Origin.X,
			Y: p.anchor.Y - p.meta.Origin.Y,
		}
		if err := drawPart(canvas, p.assetsPath(req.AssetsRoot), blitTopLeft); err != nil {
			return err
		}
	}
	return nil
}

// assetsPath returns the absolute on-disk path to this placement's PNG.
func (p placement) assetsPath(assetsRoot string) string {
	return filepath.Join(
		assetsRoot, "character-parts", p.templateId,
		p.stance, strconv.Itoa(p.frame), p.partName+".png",
	)
}

// appendBodyParts seeds the placement list with the body skin's parts. The
// `body` part anchors at (CW/2, FootRow-4); every other part of the body img
// joins via a shared joint (typically `arm` joins via `navel`).
func (c *Compositor) appendBodyParts(placed *[]placement, assetsRoot, templateId, stance string, frame int) error {
	parts, err := listFrameParts(assetsRoot, templateId, stance, frame)
	if err != nil {
		return err
	}
	bodyAnchor := Anchor{X: CanvasWidth / 2, Y: FootRow - 4}

	// Place body first if present — it seeds the chain.
	bodyMeta, hasBody := loadOrEmpty(assetsRoot, templateId, stance, frame, "body")
	if hasBody {
		*placed = append(*placed, placement{
			templateId: templateId,
			stance:     stance,
			frame:      frame,
			partName:   "body",
			meta:       bodyMeta,
			anchor:     bodyAnchor,
		})
	}

	for _, name := range parts {
		if name == "body" {
			continue
		}
		meta, ok := loadOrEmpty(assetsRoot, templateId, stance, frame, name)
		if !ok {
			continue
		}
		anchor, found := solveViaSharedJoint(*placed, meta)
		if !found {
			// First non-body part with no body match: fall back to body anchor
			// so the body img still renders in synthetic fixtures that omit
			// joint metadata.
			if !hasBody {
				anchor = bodyAnchor
			} else {
				continue
			}
		}
		*placed = append(*placed, placement{
			templateId: templateId,
			stance:     stance,
			frame:      frame,
			partName:   name,
			meta:       meta,
			anchor:     anchor,
		})
	}
	return nil
}

// appendTemplateParts adds every part of a non-equipment template (today: the
// head template) to the placement list, anchoring each via shared joints
// against parts already placed.
//
// `requireParent` — when true, skip parts that fail to find a parent joint
// (orphan) instead of failing. The head template parts must all match to
// the body's `neck` joint, so requireParent=true is fine here.
func (c *Compositor) appendTemplateParts(placed *[]placement, assetsRoot, templateId, stance string, frame int, requireParent bool) error {
	resolvedStance, resolvedFrame, err := resolveTemplateStance(assetsRoot, templateId, stance, frame)
	if err != nil {
		if errors.Is(err, ErrAssetsMissing) {
			// Head template missing is unusual but not fatal — just skip and
			// let the body render alone (some fixtures don't include heads).
			return nil
		}
		return err
	}
	parts, err := listFrameParts(assetsRoot, templateId, resolvedStance, resolvedFrame)
	if err != nil {
		return err
	}
	for _, name := range parts {
		meta, ok := loadOrEmpty(assetsRoot, templateId, resolvedStance, resolvedFrame, name)
		if !ok {
			continue
		}
		anchor, found := solveViaSharedJoint(*placed, meta)
		if !found {
			if requireParent {
				continue
			}
			anchor = Anchor{X: CanvasWidth / 2, Y: FootRow - 4}
		}
		*placed = append(*placed, placement{
			templateId: templateId,
			stance:     resolvedStance,
			frame:      resolvedFrame,
			partName:   name,
			meta:       meta,
			anchor:     anchor,
		})
	}
	return nil
}

// appendEquipmentParts is a thin wrapper for equipment slot ids that resolves
// the on-disk template directory name and reuses appendTemplateParts.
func (c *Compositor) appendEquipmentParts(placed *[]placement, req CompositeRequest, templateNumeric int, stance string) error {
	return c.appendTemplateParts(placed, req.AssetsRoot, strconv.Itoa(templateNumeric), stance, req.Frame, true)
}

// solveViaSharedJoint walks the placed parts in reverse (most-recent first)
// looking for any joint name that exists in both `part.Map` and a placed
// part's map. Reverse iteration handles the chain naturally — for example a
// weapon (with `hand`) will find arm (most-recent placement with `hand`)
// before falling back to body.
//
// Returns (anchor, true) on a match, (zero, false) for orphans.
func solveViaSharedJoint(placed []placement, part PartMeta) (Anchor, bool) {
	for jointName, childJoint := range part.Map {
		for i := len(placed) - 1; i >= 0; i-- {
			parentJoint, ok := placed[i].meta.Map[jointName]
			if !ok {
				continue
			}
			return Anchor{
				X: placed[i].anchor.X + parentJoint.X - childJoint.X,
				Y: placed[i].anchor.Y + parentJoint.Y - childJoint.Y,
			}, true
		}
	}
	return Anchor{}, false
}

// zIndex returns the position of z in the zmap slice. Unknown values sort to
// the back so they appear on top of everything.
func zIndex(zmap []string, z string) int {
	for i, name := range zmap {
		if strings.EqualFold(name, z) {
			return i
		}
	}
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

// drawPart blits the PNG at `path` onto the canvas with its top-left at
// `topLeft`.
func drawPart(canvas *image.RGBA, path string, topLeft Anchor) error {
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
	dr := image.Rect(topLeft.X, topLeft.Y, topLeft.X+w, topLeft.Y+h)
	draw.Draw(canvas, dr, img, image.Point{}, draw.Over)
	return nil
}

// resolveTemplateStance returns the stance and frame to use when looking up
// assets for an equipment template. Equipment that doesn't animate (hair,
// face, hats, glasses, etc.) only has assets under default/0. When the
// requested stance/frame is missing we fall back through default/0 then any
// available "stand" stance so items still render even when the WZ source
// only ships one stance variant — most commonly crossbows, knuckles, and
// guns whose stand2 frames are missing from the extract even though their
// owners (Bowmaster/Marksman, Buccaneer, Corsair) get forced to stand2 by
// the two-handed override.
//
// Body skins always have proper stance dirs and should NOT use this helper.
func resolveTemplateStance(assetsRoot, templateId, stance string, frame int) (string, int, error) {
	if _, err := listFrameParts(assetsRoot, templateId, stance, frame); err == nil {
		return stance, frame, nil
	} else if !errors.Is(err, ErrAssetsMissing) {
		return "", 0, err
	}
	if _, err := listFrameParts(assetsRoot, templateId, "default", 0); err == nil {
		return "default", 0, nil
	}
	// Last-chance fallback: try the other "stand" stance. Joint maps for
	// stand1/stand2 share names (hand, navel, etc.) so the part still
	// anchors via solveViaSharedJoint, even if the pose is slightly off.
	for _, alt := range stanceFallbacks(stance) {
		if _, err := listFrameParts(assetsRoot, templateId, alt, frame); err == nil {
			return alt, frame, nil
		}
	}
	return "", 0, fmt.Errorf("%w: %s/%s/%d (no default, no stand fallback)", ErrAssetsMissing, templateId, stance, frame)
}

// stanceFallbacks returns alternate stance directories to probe when the
// requested stance is missing from an equipment template. Order matters:
// stand2 → stand1 first (the common bow/gun/knuckle case), with walk1 and
// alert as further degradations to keep weapons rendering even on partial
// extracts.
func stanceFallbacks(stance string) []string {
	switch stance {
	case "stand2":
		return []string{"stand1", "walk1", "alert"}
	case "stand1":
		return []string{"stand2", "walk1", "alert"}
	default:
		return []string{"stand1", "stand2"}
	}
}

// mustSkin returns the WZ id for a validated internal skin (caller ensures
// validity via MapInternalSkin upstream).
func mustSkin(internal int) int {
	id, _ := MapInternalSkin(internal)
	return id
}
