package character

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"sort"
	"strings"

	"atlas-renders/storage"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
	"github.com/sirupsen/logrus"
)

// Native compositing canvas. Mirrors the donor's characterimage constants so
// renders position the body skin at the same `(CanvasWidth/2, FootRow-4)`
// origin used by atlas-wz-extractor. Donor: characterimage/compositor.go:19-23.
const (
	CanvasWidth  = 96
	CanvasHeight = 128
	FootRow      = 124
)

// internalSkinToWZ maps the atlas-ui internal 0..10 to the Character.wz id
// 2000..2013. Donor: characterimage/skin.go:5-19.
var internalSkinToWZ = map[int]int{
	0: 2000, 1: 2001, 2: 2002, 3: 2003, 4: 2004, 5: 2005,
	6: 2009, 7: 2010, 8: 2011, 9: 2012, 10: 2013,
}

// supportedStances is the donor's allow-list. Donor: characterimage/stance.go.
var supportedStances = map[string]struct{}{
	"stand1": {}, "stand2": {}, "walk1": {}, "alert": {}, "jump": {},
}

// SupportedStances returns the canonical list (used for error meta).
func SupportedStances() []string {
	return []string{"stand1", "stand2", "walk1", "alert", "jump"}
}

// Equipment slots that v1 silently drops before compositing. Donor:
// characterimage/filter.go.
var droppedSlots = map[int]struct{}{
	-14: {},
	-18: {}, -19: {}, -20: {},
	-21: {}, -22: {}, -23: {}, -24: {}, -25: {},
	-26: {}, -27: {}, -28: {}, -29: {}, -30: {},
}

// isTwoHandedItem reports whether an equipped weapon id forces the stand2
// stance. Delegates to libs/atlas-constants/item.IsTwoHanded so the
// authoritative weapon-classification taxonomy lives in one place. Returns
// false for any non-weapon id (the input check the library makes).
//
// Note: the previous inline implementation treated classification 147
// (Claws) as two-handed; that was an inadvertent over-inclusion. v83 claws
// are one-handed; item.IsTwoHanded reflects that.
func isTwoHandedItem(id int) bool {
	if id <= 0 {
		return false
	}
	return item.IsTwoHanded(item.Id(uint32(id)))
}

// partClassFor returns the atlas-renders MinIO partClass for a v83 item id.
// Switch keys off the typed item.Classification rather than ad-hoc
// /10000 arithmetic so the categories stay in lockstep with the shared
// taxonomy in libs/atlas-constants/item.
//
// The partClass strings ("Cap", "Coat", "Weapon", …) are atlas-renders
// MinIO directory names and stay local; only the classification side is
// delegated.
func partClassFor(id int) (string, bool) {
	if id <= 0 {
		return "", false
	}
	c := item.GetClassification(item.Id(uint32(id)))
	switch c {
	case item.ClassificationHat:
		return "Cap", true
	case item.ClassificationFaceAccessory:
		return "FaceAccessory", true
	case item.ClassificationEyeAccessory:
		return "EyeAccessory", true
	case item.ClassificationEarring:
		return "Earrings", true
	case item.ClassificationTop:
		return "Coat", true
	case item.ClassificationOverall:
		return "Longcoat", true
	case item.ClassificationBottom:
		return "Pants", true
	case item.ClassificationShoes:
		return "Shoes", true
	case item.ClassificationGloves:
		return "Glove", true
	case item.ClassificationShield:
		return "Shield", true
	case item.ClassificationCape,
		item.ClassificationRing,
		item.ClassificationPendant,
		item.ClassificationBelt,
		item.ClassificationMedal:
		return "Cape", true
	}
	// Weapons span classifications 130-149 inclusive; the shared library
	// represents this range via WeaponType. Anything that has a known
	// weapon type is a weapon for compositing purposes.
	if item.GetWeaponType(item.Id(uint32(id))) != item.WeaponTypeNone {
		return "Weapon", true
	}
	return "", false
}

// hairPartClass / facePartClass / bodyPartClass are constants so call sites
// can avoid string literals for the WZ-class subtrees that are never
// indirected through partClassFor.
const (
	hairPartClass = "Hair"
	facePartClass = "Face"
	bodyPartClass = "Body"
)

// topSlot / bottomSlot are the synthetic equipment-slot integers for the coat
// (Top/Overall) and pants (Bottom) halves, matching slotForItemID's numbering.
const (
	topSlot    = -5
	bottomSlot = -6
)

// applyDefaultClothing fills empty clothing slots with the gender's beginner
// coat/pants so a character is never rendered bare (PRD FR-2). An equipped
// Overall in the top slot covers both halves and suppresses both defaults. The
// two slots are otherwise independent. Defaults are injected as ordinary slot
// entries, so they flow through the existing compositing path unchanged.
func applyDefaultClothing(equipment map[int]int, gender int) {
	if id, ok := equipment[topSlot]; ok &&
		item.GetClassification(item.Id(uint32(id))) == item.ClassificationOverall {
		return
	}
	if _, ok := equipment[topSlot]; !ok {
		equipment[topSlot] = defaultCoat(gender)
	}
	if _, ok := equipment[bottomSlot]; !ok {
		equipment[bottomSlot] = defaultPants(gender)
	}
}

// FilterEquipment returns a copy of `in` with mount/pet/cash slots removed.
// Donor: characterimage/filter.go:15-27.
func FilterEquipment(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for slot, id := range in {
		if _, dropped := droppedSlots[slot]; dropped {
			continue
		}
		if slot <= -101 && slot >= -114 {
			continue
		}
		out[slot] = id
	}
	return out
}

// ItemsToSlotMap collapses the sorted item list into a slot→id map, mirroring
// the donor handler's itemsToSlotMap. Items in classification ranges outside
// the donor's whitelist are silently dropped.
//
// Donor: characterrender/handler.go:196-235.
func ItemsToSlotMap(items []int) map[int]int {
	out := map[int]int{}
	for _, id := range items {
		slot, ok := slotForItemID(id)
		if !ok {
			continue
		}
		out[slot] = id
	}
	return out
}

// slotForItemID maps an equipped item id to its canonical equipment-slot
// integer. The synthetic negative slot scheme follows the v83 client
// convention; -11 is the weapon slot and is the signal compositing code
// inspects to decide stand2 vs stand1.
//
// The classification side switches on the typed item.Classification from
// libs/atlas-constants/item; the slot numbers themselves are atlas-renders
// concerns that stay local. Anything with a known WeaponType lands on the
// weapon slot regardless of subclass (1H / 2H / claw / gun / etc).
func slotForItemID(id int) (int, bool) {
	if id <= 0 {
		return 0, false
	}
	c := item.GetClassification(item.Id(uint32(id)))
	switch c {
	case item.ClassificationHat:
		return -1, true
	case item.ClassificationFaceAccessory:
		return -2, true
	case item.ClassificationEyeAccessory:
		return -3, true
	case item.ClassificationEarring:
		return -4, true
	case item.ClassificationTop, item.ClassificationOverall:
		return -5, true
	case item.ClassificationBottom:
		return -6, true
	case item.ClassificationShoes:
		return -7, true
	case item.ClassificationGloves:
		return -8, true
	case item.ClassificationShield:
		return -10, true
	case item.ClassificationCape,
		item.ClassificationRing,
		item.ClassificationPendant,
		item.ClassificationBelt,
		item.ClassificationMedal:
		return -9, true
	}
	if item.GetWeaponType(item.Id(uint32(id))) != item.WeaponTypeNone {
		return -11, true
	}
	return 0, false
}

// MapInternalSkin returns the WZ skin id for an internal 0..10 value.
// Donor: characterimage/skin.go:22-27.
func MapInternalSkin(internal int) (int, error) {
	if wz, ok := internalSkinToWZ[internal]; ok {
		return wz, nil
	}
	return 0, fmt.Errorf("%w: %d (must be 0..10)", ErrUnknownSkin, internal)
}

// ValidateStance returns ErrInvalidStance if `s` is not in scope.
func ValidateStance(s string) error {
	if _, ok := supportedStances[s]; ok {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidStance, s)
}

// placement is a positioned, ready-to-draw sprite. The compositor records the
// canvas anchor at which `sprite.Origin` should land; the actual blit then
// happens at `(anchor - origin)`. Donor: characterimage/compositor.go:168-177.
type placement struct {
	templateID  uint32
	partClass   string
	sprite      manifest.Sprite
	atlasImage  image.Image
	anchor      anchorPoint
}

// anchorPoint is the canvas-space coordinate at which a sprite's `origin`
// lands. Donor: characterimage/joints.go:6.
type anchorPoint struct{ X, Y int }

// Composite walks the loadout, fetches each part atlas + manifest from MinIO,
// solves shared-joint anchors against already-placed parts, sorts by Z, and
// blits the resulting subrects onto a fresh NRGBA canvas. The output is the
// raw composited image; the caller is responsible for nearest-neighbor
// upscaling (resize) and PNG encoding.
//
// The two-handed-weapon stance override (donor: characterimage/stance.go) is
// applied here: if a 2H weapon is equipped and that weapon's manifest ships a
// stand2 sprite, the rendered stance becomes stand2 regardless of the
// requested value.
func Composite(ctx context.Context, l logrus.FieldLogger, s *storage.Storage, t tenant.Model, q RenderQuery) (image.Image, string, bool, error) {
	if err := ValidateStance(q.Stance); err != nil {
		return nil, "", false, err
	}
	wzSkin, err := MapInternalSkin(q.Skin)
	if err != nil {
		return nil, "", false, err
	}

	region := t.Region()
	version := fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())
	tenantID := t.Id().String()

	equipment := FilterEquipment(ItemsToSlotMap(q.Items))
	applyDefaultClothing(equipment, ResolveGender(q.Gender, q.Face))

	// 1. Two-handed stance override. We must resolve this BEFORE building any
	//    placements so the body skin pulls the right stance frames.
	stance := q.Stance
	twoHandedOverride := false
	if weaponID, ok := equipment[-11]; ok && isTwoHandedItem(weaponID) {
		if weaponHasStand2(ctx, l, s, tenantID, region, version, uint32(weaponID)) {
			if stance != "stand2" {
				twoHandedOverride = true
			}
			stance = "stand2"
		}
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))
	placements := make([]placement, 0, 32)
	// owners records (templateID, vslot, ownerKind) for every placed template
	// so applyVslotOcclusion below can resolve cross-template slot conflicts
	// (e.g. a full helmet's "CpH1H2H3H4H5HfHsHbAe" claim suppressing hair
	// parts mapped to those codes via the smap).
	owners := make([]vslotOwner, 0, 8)

	// 2. Body skin. The body img id is the WZ skin id directly (e.g. 2000
	//    for female skin 0). Heads live under skin+10000 (12000 for skin 0).
	bodyID := uint32(wzSkin)
	headID := uint32(wzSkin + 10000)
	bodyAnchor := anchorPoint{X: CanvasWidth / 2, Y: FootRow - 4}

	bodyAtlas, err := fetchAtlas(ctx, s, tenantID, region, version, bodyPartClass, bodyID)
	if err != nil {
		return nil, "", false, fmt.Errorf("%w: body skin %d", ErrAssetMissing, bodyID)
	}

	// Validate that the requested frame exists for the body's stance.
	if !atlasHasFrame(bodyAtlas.Manifest, stance, q.Frame) {
		return nil, "", false, fmt.Errorf("%w: body=%d stance=%s frame=%d", ErrFrameOutOfRange, bodyID, stance, q.Frame)
	}

	if err := appendBodyParts(&placements, bodyAtlas, bodyID, stance, q.Frame, bodyAnchor); err != nil {
		return nil, "", false, err
	}
	owners = appendOwner(owners, bodyID, bodyAtlas.Manifest.Vslot, ownerBody)

	// 3. Head template. The head atlas always renders at `front/0` per donor
	//    appendTemplateParts (compositor.go:194-199). It joins via the shared
	//    `neck` joint against the body parts already placed.
	if headAtlas, herr := fetchAtlas(ctx, s, tenantID, region, version, bodyPartClass, headID); herr == nil {
		_ = appendTemplateParts(&placements, headAtlas, headID, bodyPartClass, "front", 0, true, bodyAnchor)
		owners = appendOwner(owners, headID, headAtlas.Manifest.Vslot, ownerHead)
	} else {
		l.WithError(herr).Debugf("no head atlas for skin id %d (continuing)", headID)
	}

	// 4. Equipment in iteration order — z-ordering happens after all parts are
	//    placed, so insertion order is only used for the joint-graph chain.
	for _, id := range equipment {
		if id == 0 {
			continue
		}
		pc, ok := partClassFor(id)
		if !ok {
			continue
		}
		atl, ferr := fetchAtlas(ctx, s, tenantID, region, version, pc, uint32(id))
		if ferr != nil {
			l.WithError(ferr).Warnf("missing atlas: partClass=%s id=%d (skipping)", pc, id)
			continue
		}
		rstance, rframe := resolveTemplateStance(atl.Manifest, stance, q.Frame)
		_ = appendTemplateParts(&placements, atl, uint32(id), pc, rstance, rframe, true, bodyAnchor)
		owners = appendOwner(owners, uint32(id), atl.Manifest.Vslot, ownerEquipment)
	}

	// 5. Hair + face. Hair attaches via the head's brow joint; face uses
	//    earOverHead. Both are emitted as equipment-style placements.
	if q.Hair != 0 {
		if atl, ferr := fetchAtlas(ctx, s, tenantID, region, version, hairPartClass, uint32(q.Hair)); ferr == nil {
			rstance, rframe := resolveTemplateStance(atl.Manifest, stance, q.Frame)
			_ = appendTemplateParts(&placements, atl, uint32(q.Hair), hairPartClass, rstance, rframe, true, bodyAnchor)
			owners = appendOwner(owners, uint32(q.Hair), atl.Manifest.Vslot, ownerHair)
		} else {
			l.WithError(ferr).Warnf("missing hair atlas id=%d", q.Hair)
		}
	}
	if q.Face != 0 {
		if atl, ferr := fetchAtlas(ctx, s, tenantID, region, version, facePartClass, uint32(q.Face)); ferr == nil {
			rstance, rframe := resolveTemplateStance(atl.Manifest, stance, q.Frame)
			_ = appendTemplateParts(&placements, atl, uint32(q.Face), facePartClass, rstance, rframe, true, bodyAnchor)
			owners = appendOwner(owners, uint32(q.Face), atl.Manifest.Vslot, ownerFace)
		} else {
			l.WithError(ferr).Warnf("missing face atlas id=%d", q.Face)
		}
	}

	// 6. Character-meta sidecars (smap + zmap). They are co-located under the
	//    same scope and emitted together by the atlas-data Character worker,
	//    so one scope probe serves both.
	//    - smap drives vslot occlusion (a full helmet's slot claim suppresses
	//      hair parts mapped to those codes). On failure we log and continue
	//      without occlusion (bangs paint over a full helmet — pre-task
	//      baseline). Owners must be precedence-sorted (equipment < hair <
	//      face < head < body) before applyVslotOcclusion runs.
	//    - zmap is the ordered layer list that drives part z-order in step 7.
	//      On failure z-ordering falls back to insertion order.
	//    Donor: characterimage/compositor.go:232-235.
	var zmap []string
	metaScope, scopeErr := s.ResolveSmapScope(ctx, tenantID, region, version)
	if scopeErr != nil {
		l.WithError(scopeErr).Warn("character-meta scope resolve failed; vslot occlusion and zmap z-ordering disabled")
	} else {
		if smap, smapErr := s.GetSmap(ctx, metaScope, region, version); smapErr != nil {
			l.WithError(smapErr).Warn("smap fetch failed; vslot occlusion disabled (full helmets will not hide bangs)")
		} else {
			sort.SliceStable(owners, func(i, j int) bool { return owners[i].kind < owners[j].kind })
			placements = applyVslotOcclusion(placements, smap, owners)
		}
		if zm, zmapErr := s.GetZmap(ctx, metaScope, region, version); zmapErr != nil {
			l.WithError(zmapErr).Warn("zmap fetch failed; character z-ordering falls back to insertion order (weapons/shields/accessories may layer wrong)")
		} else {
			zmap = zm
		}
	}

	// 7. Sort by zmap render order. zmap is front-to-back (index 0 = most
	//    frontward), so the back-most layer — highest index, and unknown
	//    layers which zIndex maps to len(zmap) — must draw first. The sort key
	//    is sprite.Z, the WZ render-layer label (e.g. "weaponOverGlove"), NOT
	//    sprite.Part (the canvas name, often generic like "weapon"): the layer
	//    a sprite occupies varies by stance/frame and is carried by the `z`
	//    child, so keying on Part mislayers weapons/shields/etc. When zmap is
	//    empty (sidecar missing) every part resolves to index 0 and the stable
	//    sort preserves insertion order. Donor:
	//    characterimage/compositor.go:240-242 (zIndex(zmap, meta.Z)).
	sort.SliceStable(placements, func(i, j int) bool {
		return zIndex(zmap, string(placements[i].sprite.Z)) > zIndex(zmap, string(placements[j].sprite.Z))
	})

	// 8. Blit each placement at `(anchor - origin)` top-left, cropping the
	//    sprite's subrect out of the per-templateId atlas image. Donor:
	//    characterimage/compositor.go:244-252.
	for _, p := range placements {
		srcR := image.Rect(p.sprite.Rect.X, p.sprite.Rect.Y,
			p.sprite.Rect.X+p.sprite.Rect.W, p.sprite.Rect.Y+p.sprite.Rect.H)
		dx := p.anchor.X - p.sprite.Origin.X
		dy := p.anchor.Y - p.sprite.Origin.Y
		dstR := image.Rect(dx, dy, dx+p.sprite.Rect.W, dy+p.sprite.Rect.H)
		draw.Draw(canvas, dstR, p.atlasImage, srcR.Min, draw.Over)
	}

	return canvas, stance, twoHandedOverride, nil
}

// appendBodyParts seeds the placement list with the body skin's parts at the
// resolved stance/frame. The `body` part anchors at the canvas bodyAnchor;
// every other part of the body atlas joins via a shared joint with an
// already-placed part. Donor: characterimage/compositor.go:267-316.
func appendBodyParts(placed *[]placement, atl *storage.AtlasEntry, templateID uint32, stance string, frame int, bodyAnchor anchorPoint) error {
	img, err := png.Decode(bytes.NewReader(atl.PNG))
	if err != nil {
		return fmt.Errorf("decode body atlas: %w", err)
	}

	// Index sprites for this (stance, frame).
	byPart := make(map[string]manifest.Sprite)
	parts := []string{}
	for _, sp := range atl.Manifest.Sprites {
		if sp.Stance != stance || sp.Frame != frame {
			continue
		}
		byPart[sp.Part] = sp
		parts = append(parts, sp.Part)
	}

	// Place `body` first so subsequent parts can resolve against it.
	if body, ok := byPart["body"]; ok {
		*placed = append(*placed, placement{
			templateID: templateID,
			partClass:  bodyPartClass,
			sprite:     body,
			atlasImage: img,
			anchor:     bodyAnchor,
		})
		delete(byPart, "body")
	}

	for _, name := range parts {
		if name == "body" {
			continue
		}
		sp, ok := byPart[name]
		if !ok {
			continue
		}
		anchor, found := solveViaSharedJoint(*placed, sp)
		if !found {
			// Synthetic fixtures may omit joint metadata. Fall back to the
			// body anchor so the body atlas still renders; matches donor
			// behaviour at compositor.go:300-303.
			anchor = bodyAnchor
		}
		*placed = append(*placed, placement{
			templateID: templateID,
			partClass:  bodyPartClass,
			sprite:     sp,
			atlasImage: img,
			anchor:     anchor,
		})
	}
	return nil
}

// appendTemplateParts adds every sprite of a non-body atlas at the resolved
// (stance, frame) to the placement list, anchoring each via shared joints
// against parts already placed. When `requireParent` is true, parts that fail
// joint resolution are dropped (the donor's behaviour for the head template
// and equipment). Donor: characterimage/compositor.go:325-361.
func appendTemplateParts(placed *[]placement, atl *storage.AtlasEntry, templateID uint32, partClass, stance string, frame int, requireParent bool, bodyAnchor anchorPoint) error {
	img, err := png.Decode(bytes.NewReader(atl.PNG))
	if err != nil {
		return fmt.Errorf("decode atlas %s/%d: %w", partClass, templateID, err)
	}
	for _, sp := range atl.Manifest.Sprites {
		if sp.Stance != stance || sp.Frame != frame {
			continue
		}
		anchor, found := solveViaSharedJoint(*placed, sp)
		if !found {
			if requireParent {
				continue
			}
			anchor = bodyAnchor
		}
		*placed = append(*placed, placement{
			templateID: templateID,
			partClass:  partClass,
			sprite:     sp,
			atlasImage: img,
			anchor:     anchor,
		})
	}
	return nil
}

// solveViaSharedJoint walks placed parts in reverse (most-recent first) looking
// for any joint name that exists in both the candidate sprite's Anchors map
// and a placed sprite's Anchors map. Reverse iteration is what produces the
// natural chain: a weapon's `hand` joint will find an arm before falling back
// to the body. Donor: characterimage/compositor.go:376-390.
func solveViaSharedJoint(placed []placement, child manifest.Sprite) (anchorPoint, bool) {
	for jointName, childJoint := range child.Anchors {
		for i := len(placed) - 1; i >= 0; i-- {
			parent := placed[i].sprite
			parentJoint, ok := parent.Anchors[jointName]
			if !ok {
				continue
			}
			return anchorPoint{
				X: placed[i].anchor.X + parentJoint.X - childJoint.X,
				Y: placed[i].anchor.Y + parentJoint.Y - childJoint.Y,
			}, true
		}
	}
	return anchorPoint{}, false
}

// resolveTemplateStance returns the stance/frame to use for a non-body atlas.
// Equipment that doesn't animate (hair, face, hats, glasses, etc.) only has
// sprites under default/0. When the requested stance/frame is missing we fall
// back through default/0 then any available stand stance so items still
// render even when ingest only emitted one stance variant.
//
// Donor: characterimage/compositor.go:459-477.
func resolveTemplateStance(m manifest.Manifest, stance string, frame int) (string, int) {
	if atlasHasFrame(m, stance, frame) {
		return stance, frame
	}
	if atlasHasFrame(m, "default", 0) {
		return "default", 0
	}
	for _, alt := range stanceFallbacks(stance) {
		if atlasHasFrame(m, alt, frame) {
			return alt, frame
		}
	}
	// Last resort — let the caller's empty-filter loop pick up nothing.
	return stance, frame
}

// stanceFallbacks returns alternate stance directories to probe when the
// requested stance is missing. Donor: characterimage/compositor.go:484-493.
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

// atlasHasFrame returns true if the manifest contains at least one sprite
// matching (stance, frame).
func atlasHasFrame(m manifest.Manifest, stance string, frame int) bool {
	for _, sp := range m.Sprites {
		if sp.Stance == stance && sp.Frame == frame {
			return true
		}
	}
	return false
}

// weaponHasStand2 returns true if the weapon's MinIO atlas ships any stand2
// sprite. The donor probes the on-disk stand2/0 dir; we probe the manifest
// since atlas-renders has no filesystem access. Donor: characterimage/
// stance.go:43-52.
func weaponHasStand2(ctx context.Context, l logrus.FieldLogger, s *storage.Storage, tenantID, region, version string, weaponID uint32) bool {
	atl, err := fetchAtlas(ctx, s, tenantID, region, version, "Weapon", weaponID)
	if err != nil {
		l.WithError(err).Debugf("two-handed probe miss for weapon %d", weaponID)
		return false
	}
	for _, sp := range atl.Manifest.Sprites {
		if sp.Stance == "stand2" {
			return true
		}
	}
	return false
}

// fetchAtlas resolves the per-(partClass, tenant) scope and fetches the atlas
// + manifest. Both lookups are LRU-backed via Storage so the cost amortises
// across renders. ResolveScope takes the full bucket subpath; character
// atlases live under "atlases/<partClass>/" so we pass that prefix.
func fetchAtlas(ctx context.Context, s *storage.Storage, tenantID, region, version, partClass string, id uint32) (*storage.AtlasEntry, error) {
	scope, err := s.ResolveScope(ctx, tenantID, region, version, "atlases/"+partClass)
	if err != nil {
		return nil, fmt.Errorf("resolve scope %s: %w", partClass, err)
	}
	return s.GetAtlas(ctx, scope, region, version, partClass, id)
}

// zIndex returns the render-order index of a layer name within zmap, matching
// case-insensitively (shipped zmap.img entries are camelCase; the smap and
// synthetic fixtures sometimes lower-case). zmap is front-to-back, so a lower
// index is more frontward. Unknown layers return len(zmap) so an unmapped part
// sorts to the back and never paints over a mapped one. An empty zmap maps
// every layer to 0, collapsing the sort to insertion order (the graceful
// fallback when the zmap.json sidecar is absent). Donor:
// characterimage/compositor.go zIndex().
func zIndex(zmap []string, layer string) int {
	for i, name := range zmap {
		if strings.EqualFold(name, layer) {
			return i
		}
	}
	return len(zmap)
}

// pickSprite returns the first manifest sprite matching (stance, frame, part)
// or nil. Exposed for tests; the compositor itself iterates manifest.Sprites
// directly to avoid the indirection.
func pickSprite(m manifest.Manifest, stance string, frame int, part string) *manifest.Sprite {
	for i := range m.Sprites {
		sp := &m.Sprites[i]
		if sp.Stance == stance && sp.Frame == frame && sp.Part == part {
			return sp
		}
	}
	return nil
}

// NearestNeighborUpscale produces an integer-multiple upscale of src using
// nearest-neighbor sampling so each source pixel becomes an N×N block.
// Donor: characterimage/scale.go:9-23.
func NearestNeighborUpscale(src image.Image, resize int) image.Image {
	if resize < 1 {
		resize = 1
	}
	if resize == 1 {
		return src
	}
	sb := src.Bounds()
	w, h := sb.Dx()*resize, sb.Dy()*resize
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y/resize
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x/resize
			out.Set(x, y, src.At(sx, sy))
		}
	}
	return out
}

