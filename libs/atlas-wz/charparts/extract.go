// Package charparts ports the donor atlas-wz-extractor character-parts
// walker into an io-agnostic library. Where the donor wrote per-template
// PNGs + sidecar JSON to a filesystem hierarchy, this package decodes
// canvases into image.Image values held in memory so callers can feed them
// straight into atlas.Pack and stream the resulting sheet/manifest pair to
// any sink (MinIO, byte buffer, disk).
//
// The library does NOT call atlas.Pack itself — callers control packing so
// they can choose how to scope partClass/id batches and what to do with
// individual decode failures.
//
// Donor: services/atlas-wz-extractor/atlas.com/wz-extractor/image/
// character_parts.go (511 LOC); donor extract.go for findSub / normalizeId.
package charparts

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/atlas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/canvas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// PartSet is one (partClass, id) pack-ready bundle. Sprites carries the
// per-(stance,frame,part) decoded canvases the caller feeds into atlas.Pack;
// Info carries the .img-level metadata the donor emitted as info.json.
type PartSet struct {
	PartClass string
	ID        uint32
	Sprites   []SpriteInput
	Info      InfoSidecar
}

// SpriteInput is one decoded sprite ready to be packed. Stance/Frame/Part
// are the donor's path tags ({stance}/{frame}/{part}.png in the original
// filesystem layout); callers preserve them in the resulting manifest so
// renderers can index by (stance, frame, part).
type SpriteInput struct {
	Stance  string
	Frame   int
	Part    string
	Img     image.Image
	Origin  image.Point
	Anchors map[string]image.Point
	Z       int
}

// InfoSidecar mirrors the donor's templateInfo block. Vslot is the field
// atlas-renders needs for equipment-vs-hair occlusion; Islot and Cash are
// kept for downstream consumers (and to match the donor's writeInfoJSON
// payload verbatim).
type InfoSidecar struct {
	Islot string
	Vslot string
	Cash  int
}

// equipmentSubdirs enumerates the Character.wz directories whose .img files
// the donor materialized as worn sprites. Body skin imgs (0000xxxx / 0001xxxx)
// live at the root of Character.wz — they have no subdirectory entry and are
// handled separately by WalkCharacter. Donor: character_parts.go:75-78.
var equipmentSubdirs = []string{
	"Cap", "Coat", "Longcoat", "Pants", "Shoes", "Glove",
	"Cape", "Shield", "Weapon", "Hair", "Face", "Accessory",
}

// subdirToPartClass maps a Character.wz subdirectory name to the atlas-renders
// partClass string. For most subdirs the mapping is identity; "Accessory"
// expands at extraction time because v83 stores caps/masks/glasses/earrings
// under a single Accessory directory keyed by the .img's id range.
//
// The "Body" partClass is used for both body skin .img (id 2000..2013) and
// head .img (id 12000..12013); both are at the root of Character.wz.
// atlas-renders' composite.go bodyPartClass="Body" constant pairs with this.
var subdirToPartClass = map[string]string{
	"Cap":      "Cap",
	"Coat":     "Coat",
	"Longcoat": "Longcoat",
	"Pants":    "Pants",
	"Shoes":    "Shoes",
	"Glove":    "Glove",
	"Cape":     "Cape",
	"Shield":   "Shield",
	"Weapon":   "Weapon",
	"Hair":     "Hair",
	"Face":     "Face",
}

// accessoryPartClassFor classifies an Accessory subdirectory .img by its id
// range. v83 Character.wz/Accessory stores eye/face/earring accessories under
// the single dir; atlas-renders splits them via classifications 101xxxx
// (FaceAccessory), 102xxxx (EyeAccessory), 103xxxx (Earrings).
func accessoryPartClassFor(id uint32) (string, bool) {
	c := id / 10000
	switch c {
	case 101:
		return "FaceAccessory", true
	case 102:
		return "EyeAccessory", true
	case 103:
		return "Earrings", true
	}
	return "", false
}

// stancesInScope is the donor's explicit allow-list. Skipping fly/prone/swing
// keeps the sheet small. "default", "front", "back" are included for
// non-animated templates (hair/face/hats/heads). Donor: character_parts.go:53-62.
var stancesInScope = map[string]struct{}{
	"stand1":  {},
	"stand2":  {},
	"walk1":   {},
	"alert":   {},
	"jump":    {},
	"default": {},
	"front":   {},
	"back":    {},
}

// directCanvasStances are stances whose children are CanvasProperties directly
// (no frame SubProperty layer). Donor: character_parts.go:67-71.
var directCanvasStances = map[string]struct{}{
	"default": {},
	"front":   {},
	"back":    {},
}

// WalkCharacter walks a parsed Character.wz file and yields one PartSet per
// (partClass, template-id) discovered in the archive. Per-template extraction
// failures (canvas decode errors, missing data) are logged on the returned
// PartSet's Sprites being shorter than expected — the caller decides whether
// to skip a partial set or pack it anyway.
//
// partClassFilter, if non-nil, restricts the walk to PartSets whose PartClass
// is keyed in the map with value true. Passing nil walks every recognized
// partClass under the Character.wz root.
//
// The walker matches the donor's coverage:
//   - Body skin and head .img at the root (names 0000xxxx and 0001xxxx)
//     emit PartClass="Body".
//   - Each entry in equipmentSubdirs emits the partClass mapped by
//     subdirToPartClass, or for "Accessory" the partClass derived from each
//     .img's id range.
func WalkCharacter(f *wz.File, partClassFilter map[string]bool) ([]PartSet, error) {
	if f == nil {
		return nil, fmt.Errorf("charparts.WalkCharacter: nil wz.File")
	}
	root := f.Root()
	if root == nil {
		return nil, nil
	}

	want := func(pc string) bool {
		if partClassFilter == nil {
			return true
		}
		return partClassFilter[pc]
	}

	out := make([]PartSet, 0, 64)

	// Body skin (0000xxxx) + heads (0001xxxx) live at the Character.wz root.
	// Donor: character_parts.go:432-439.
	if want("Body") {
		for _, img := range root.Images() {
			if !strings.HasPrefix(img.Name(), "0000") && !strings.HasPrefix(img.Name(), "0001") {
				continue
			}
			ps, ok := extractTemplate(f, img, "Body")
			if !ok {
				continue
			}
			out = append(out, ps)
		}
	}

	// Equipment subdirectories. Donor: character_parts.go:441-453.
	for _, sub := range equipmentSubdirs {
		dir := findCharSubdir(root.Directories(), sub)
		if dir == nil {
			continue
		}
		// Accessory expands per-img into multiple partClasses.
		if sub == "Accessory" {
			for _, img := range dir.Images() {
				idStr := normalizeId(img.Name())
				idU, err := strconv.ParseUint(idStr, 10, 32)
				if err != nil {
					continue
				}
				pc, ok := accessoryPartClassFor(uint32(idU))
				if !ok || !want(pc) {
					continue
				}
				ps, ok := extractTemplate(f, img, pc)
				if !ok {
					continue
				}
				out = append(out, ps)
			}
			continue
		}
		pc, ok := subdirToPartClass[sub]
		if !ok || !want(pc) {
			continue
		}
		for _, img := range dir.Images() {
			ps, ok := extractTemplate(f, img, pc)
			if !ok {
				continue
			}
			out = append(out, ps)
		}
	}

	return out, nil
}

// extractTemplate walks one .img file and produces a PartSet. Returns
// (PartSet{}, false) if the id cannot be parsed.
//
// Donor: extractTemplateImg (character_parts.go:366-405). Differences from
// the donor:
//   - No filesystem I/O; each part canvas is decoded into image.Image.
//   - templateInfo is returned via InfoSidecar instead of written to disk.
//   - UOL targets that fail to resolve are silently skipped (donor logs and
//     continues, same effect).
func extractTemplate(f *wz.File, img *wz.Image, partClass string) (PartSet, bool) {
	idStr := normalizeId(img.Name())
	idU, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return PartSet{}, false
	}
	id := uint32(idU)

	props, err := img.Properties()
	if err != nil {
		// best-effort: skip templates whose .img cannot be parsed
		return PartSet{}, false
	}
	info := extractInfoBlock(props)
	lookup := buildPathLookup(props)

	sprites := make([]SpriteInput, 0, 32)

	for _, p := range props {
		stanceSub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		stance := stanceSub.Name()
		if _, ok := stancesInScope[stance]; !ok {
			continue
		}
		stancePath := strings.ToLower(stance)
		if _, direct := directCanvasStances[stance]; direct {
			sprites = appendDirectStanceSprites(f, sprites, stanceSub.Children(), stance, lookup, stancePath)
			continue
		}
		// Animated stance: each child is a frame SubProperty.
		for _, fp := range stanceSub.Children() {
			frameSub, ok := fp.(*property.SubProperty)
			if !ok {
				continue
			}
			frameName := frameSub.Name()
			frameIdx, err := strconv.Atoi(frameName)
			if err != nil {
				continue
			}
			framePath := stancePath + "/" + strings.ToLower(frameName)
			sprites = appendAnimatedFrameSprites(f, sprites, frameSub.Children(), stance, frameIdx, lookup, framePath)
		}
	}

	return PartSet{
		PartClass: partClass,
		ID:        id,
		Sprites:   sprites,
		Info:      info,
	}, true
}

// appendDirectStanceSprites decodes the part canvases (and UOL aliases) under
// a default/front/back stance whose children are CanvasProperty directly.
// Donor: extractDefaultStanceChildren (character_parts.go:300-328).
func appendDirectStanceSprites(f *wz.File, out []SpriteInput, children []property.Property, stance string, lookup pathLookup, stancePath string) []SpriteInput {
	const frame = 0
	for _, partProp := range children {
		switch v := partProp.(type) {
		case *property.CanvasProperty:
			sp, ok := decodePartSprite(f, v, stance, frame, v.Name())
			if !ok {
				continue
			}
			out = append(out, sp)
		case *property.UOLProperty:
			if lookup == nil {
				continue
			}
			target := resolveUOL(lookup, stancePath, v)
			cp, ok := target.(*property.CanvasProperty)
			if !ok {
				continue
			}
			sp, ok := decodePartSprite(f, cp, stance, frame, v.Name())
			if !ok {
				continue
			}
			out = append(out, sp)
		}
	}
	return out
}

// appendAnimatedFrameSprites decodes the part canvases (and UOL aliases) for
// one frame of an animated stance. Donor: extractAnimatedFrameChildren
// (character_parts.go:332-357).
func appendAnimatedFrameSprites(f *wz.File, out []SpriteInput, frameProps []property.Property, stance string, frame int, lookup pathLookup, framePath string) []SpriteInput {
	for _, partProp := range frameProps {
		switch v := partProp.(type) {
		case *property.CanvasProperty:
			sp, ok := decodePartSprite(f, v, stance, frame, v.Name())
			if !ok {
				continue
			}
			out = append(out, sp)
		case *property.UOLProperty:
			target := resolveUOL(lookup, framePath, v)
			cp, ok := target.(*property.CanvasProperty)
			if !ok {
				continue
			}
			sp, ok := decodePartSprite(f, cp, stance, frame, v.Name())
			if !ok {
				continue
			}
			out = append(out, sp)
		}
	}
	return out
}

// decodePartSprite resolves one CanvasProperty into a SpriteInput, decoding
// its pixel data and harvesting the origin/anchors/z metadata from the
// canvas children. Returns (SpriteInput{}, false) if the canvas data cannot
// be decoded.
//
// Replaces the donor's filesystem-bound writeCanvasPng + buildPartSidecar
// pipeline; the metadata fields extracted here mirror partSidecar in
// character_parts.go:21-28.
func decodePartSprite(f *wz.File, cp *property.CanvasProperty, stance string, frame int, part string) (SpriteInput, bool) {
	img, err := decodeCanvas(f, cp)
	if err != nil {
		return SpriteInput{}, false
	}
	origin, anchors, z := extractPartMetadata(cp.Children())
	return SpriteInput{
		Stance:  stance,
		Frame:   frame,
		Part:    part,
		Img:     img,
		Origin:  origin,
		Anchors: anchors,
		Z:       z,
	}, true
}

// extractPartMetadata harvests origin / anchor map / z from the children of a
// canvas property. Donor: buildPartSidecar (character_parts.go:166-203). The
// z string ("0".."10") is converted to an int so callers can sort directly.
func extractPartMetadata(children []property.Property) (image.Point, map[string]image.Point, int) {
	var origin image.Point
	anchors := map[string]image.Point{}
	z := 0
	for _, c := range children {
		switch v := c.(type) {
		case *property.VectorProperty:
			if v.Name() == "origin" {
				origin = image.Point{X: int(v.X()), Y: int(v.Y())}
			}
		case *property.StringProperty:
			if v.Name() == "z" {
				if n, err := strconv.Atoi(v.Value()); err == nil {
					z = n
				}
			}
		case *property.IntProperty:
			if v.Name() == "z" {
				z = int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == "z" {
				z = int(v.Value())
			}
		case *property.SubProperty:
			if v.Name() == "map" {
				for _, jp := range v.Children() {
					if jv, ok := jp.(*property.VectorProperty); ok {
						anchors[jv.Name()] = image.Point{X: int(jv.X()), Y: int(jv.Y())}
					}
				}
			}
		}
	}
	if len(anchors) == 0 {
		anchors = nil
	}
	return origin, anchors, z
}

// extractInfoBlock returns an InfoSidecar populated from the `info` sub of an
// equipment img. Missing fields default to zero values. Donor:
// extractInfoBlock (character_parts.go:82-108).
func extractInfoBlock(props []property.Property) InfoSidecar {
	info := findSub(props, "info")
	if info == nil {
		return InfoSidecar{}
	}
	out := InfoSidecar{}
	for _, p := range info.Children() {
		switch v := p.(type) {
		case *property.StringProperty:
			switch v.Name() {
			case "islot":
				out.Islot = v.Value()
			case "vslot":
				out.Vslot = v.Value()
			}
		case *property.IntProperty:
			if v.Name() == "cash" {
				out.Cash = int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == "cash" {
				out.Cash = int(v.Value())
			}
		}
	}
	return out
}

// pathLookup is a lower-cased absolute-path -> property index used for UOL
// resolution. Donor: pathLookup (character_parts.go:207).
type pathLookup map[string]property.Property

// buildPathLookup walks every property under root (recursively, including
// canvas children) and indexes them by their slash-joined absolute path.
// Donor: buildPathLookup (character_parts.go:215-232).
func buildPathLookup(root []property.Property) pathLookup {
	out := make(pathLookup)
	var walk func(prefix string, props []property.Property)
	walk = func(prefix string, props []property.Property) {
		for _, p := range props {
			path := strings.ToLower(p.Name())
			if prefix != "" {
				path = prefix + "/" + path
			}
			out[path] = p
			if children := p.Children(); len(children) > 0 {
				walk(path, children)
			}
		}
	}
	walk("", root)
	return out
}

// canonicalizeUOLPath resolves a UOL value relative to its anchor (the
// slash-joined absolute path of the property containing the UOL). Donor:
// canonicalizeUOLPath (character_parts.go:240-258).
func canonicalizeUOLPath(anchorPath, uolValue string) string {
	var parts []string
	if anchorPath != "" {
		parts = strings.Split(anchorPath, "/")
	}
	for _, seg := range strings.Split(uolValue, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
			continue
		}
		parts = append(parts, strings.ToLower(seg))
	}
	return strings.Join(parts, "/")
}

// resolveUOL dereferences a UOL chain (max 5 hops). Donor: resolveUOL
// (character_parts.go:265-283).
func resolveUOL(lookup pathLookup, anchorPath string, uol *property.UOLProperty) property.Property {
	current := uol
	currentAnchor := anchorPath
	for depth := 0; depth < 5; depth++ {
		target := canonicalizeUOLPath(currentAnchor, current.Value())
		resolved, ok := lookup[target]
		if !ok {
			return nil
		}
		next, isUOL := resolved.(*property.UOLProperty)
		if !isUOL {
			return resolved
		}
		currentAnchor = parentPath(target)
		current = next
	}
	return nil
}

func parentPath(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// findCharSubdir returns the child Directory whose name case-insensitively
// equals `name`, or nil. Donor: findCharSubdir (character_parts.go:504-511).
func findCharSubdir(dirs []*wz.Directory, name string) *wz.Directory {
	for _, d := range dirs {
		if strings.EqualFold(d.Name(), name) {
			return d
		}
	}
	return nil
}

// findSub returns the first SubProperty named `name` in props. Donor: findSub
// (extract.go:456-463).
func findSub(props []property.Property, name string) *property.SubProperty {
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok && sub.Name() == name {
			return sub
		}
	}
	return nil
}

// normalizeId strips leading zeros from a WZ entity name. Donor: normalizeId
// (extract.go:574-580).
func normalizeId(id string) string {
	// Strip trailing ".img" if present (image names sometimes retain the
	// extension when sourced from filesystem-style iteration).
	id = strings.TrimSuffix(id, ".img")
	trimmed := strings.TrimLeft(id, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

// decodeCanvas reads canvas data from the WZ file and decompresses it into an
// in-memory image. No filesystem I/O. Donor's defaultCanvasWriter does the
// equivalent decode then encodes to PNG on disk; we keep the decode and skip
// the write so callers can pack the raw pixels straight into atlas.Pack.
func decodeCanvas(f *wz.File, cp *property.CanvasProperty) (image.Image, error) {
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return nil, fmt.Errorf("read canvas data: %w", err)
	}
	img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("decompress canvas: %w", err)
	}
	return img, nil
}

// EncodePartName renders a (stance, frame, part) triple as the dotted form
// "stance.frame.part" that ToAtlasInputs writes into atlas.Input.Name and
// (after Pack) atlas-renders reads back via DecodePartName.
//
// The dotted form is what makes round-tripping through atlas.Pack possible:
// Pack copies Input.Name verbatim into Sprite.Part, but Pack also pre-sorts
// inputs by size, so we cannot rely on input/output index correspondence to
// re-attach stance/frame. Encoding the tags in the name preserves them across
// the sort. Donor stored these as separate filesystem path components.
func EncodePartName(stance string, frame int, part string) string {
	return fmt.Sprintf("%s.%d.%s", stance, frame, part)
}

// DecodePartName parses the dotted form produced by EncodePartName back into
// (stance, frame, part). Returns ok=false if the name is not in the expected
// shape.
func DecodePartName(name string) (stance string, frame int, part string, ok bool) {
	first := strings.IndexByte(name, '.')
	if first < 0 {
		return "", 0, "", false
	}
	second := strings.IndexByte(name[first+1:], '.')
	if second < 0 {
		return "", 0, "", false
	}
	second += first + 1
	frameN, err := strconv.Atoi(name[first+1 : second])
	if err != nil {
		return "", 0, "", false
	}
	return name[:first], frameN, name[second+1:], true
}

// ToAtlasInputs converts a PartSet's Sprites into the []atlas.Input shape
// atlas.Pack consumes. Each input's Name is set to EncodePartName(stance,
// frame, part) so the post-Pack manifest carries the dotted tag string in
// Sprite.Part; the caller is responsible for splitting it back into
// stance/frame/part fields and stamping the manifest.
func ToAtlasInputs(set PartSet) []atlas.Input {
	out := make([]atlas.Input, 0, len(set.Sprites))
	for _, s := range set.Sprites {
		if s.Img == nil {
			continue
		}
		out = append(out, atlas.Input{
			Name:    EncodePartName(s.Stance, s.Frame, s.Part),
			Img:     s.Img,
			Origin:  s.Origin,
			Anchors: s.Anchors,
			Z:       s.Z,
		})
	}
	return out
}
