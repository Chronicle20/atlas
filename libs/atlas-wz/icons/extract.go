// Package icons provides io-agnostic per-id icon extractors for NPC, mob,
// item, reactor, and skill WZ files. Each public Extract* function takes a
// parsed wz.File and the numeric id of the target entity, walks the file's
// property tree to find the appropriate canvas, and returns a decoded
// image.Image. Resolving info/link or info/icon UOL references is handled
// transparently.
//
// These functions write nothing to disk; callers route the returned image
// however they want (S3, byte buffer, http response, etc.).
package icons

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/canvas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// ErrNotFound is returned when no icon can be resolved for the requested id.
var ErrNotFound = errors.New("icons: not found")

// canvasFinder locates the appropriate per-entity canvas inside a parsed
// image's property list.
type canvasFinder func(props []property.Property) *property.CanvasProperty

// ExtractItemIcon walks an Item.wz file looking for the item with the given
// id and returns its decoded info/icon canvas. info/icon UOL references that
// point at a sibling item under the same .img are resolved.
//
// Note: the original plan signature was (img *wz.Image, id uint32), but
// items are stored across category subdirectories (Cash/Consume/Etc/Install/
// Pet) with many items per .img, so resolving a single id needs File-level
// traversal. Passing *wz.File matches the donor's actual access pattern.
func ExtractItemIcon(f *wz.File, id uint32) (image.Image, error) {
	if f == nil {
		return nil, ErrNotFound
	}
	root := f.Root()
	if root == nil {
		return nil, ErrNotFound
	}
	target := strconv.FormatUint(uint64(id), 10)

	for _, dir := range root.Directories() {
		for _, img := range dir.Images() {
			props := img.Properties()
			if len(props) == 0 {
				continue
			}

			// Single-item image (e.g. Pet): item id == img name; info/icon at root.
			if normalizeId(img.Name()) == target {
				if cp := findInfoIcon(props); cp != nil {
					return decodeCanvas(f, cp)
				}
			}

			// Multi-item image: scan top-level sub-properties for the id.
			siblings := indexItemSubs(props)
			for _, p := range props {
				sub, ok := p.(*property.SubProperty)
				if !ok {
					continue
				}
				if normalizeId(sub.Name()) != target {
					continue
				}
				cp := findInfoIcon(sub.Children())
				if cp == nil {
					cp = resolveItemIconUOL(siblings, sub.Name(), sub.Children())
				}
				if cp == nil {
					continue
				}
				return decodeCanvas(f, cp)
			}
		}
	}
	return nil, ErrNotFound
}

// ExtractNpcIcon returns the decoded stand/0 (or fallback) canvas for the
// given NPC id from a parsed Npc.wz file.
func ExtractNpcIcon(f *wz.File, id uint32) (image.Image, error) {
	return extractEntityIcon(f, id, findStandCanvas)
}

// ExtractMobIcon returns the decoded stand/0 (or fallback) canvas for the
// given mob id from a parsed Mob.wz file.
func ExtractMobIcon(f *wz.File, id uint32) (image.Image, error) {
	return extractEntityIcon(f, id, findStandCanvas)
}

// ExtractReactorIcon returns the decoded 0/0 canvas for the given reactor id
// from a parsed Reactor.wz file.
func ExtractReactorIcon(f *wz.File, id uint32) (image.Image, error) {
	return extractEntityIcon(f, id, findReactorCanvas)
}

// ExtractSkillIcon returns the decoded icon canvas for the given skill id
// from a parsed Skill.wz file.
func ExtractSkillIcon(f *wz.File, id uint32) (image.Image, error) {
	if f == nil {
		return nil, ErrNotFound
	}
	root := f.Root()
	if root == nil {
		return nil, ErrNotFound
	}
	target := strconv.FormatUint(uint64(id), 10)

	for _, img := range root.Images() {
		props := img.Properties()
		if len(props) == 0 {
			continue
		}
		skillDir := findSub(props, "skill")
		if skillDir == nil {
			continue
		}
		for _, child := range skillDir.Children() {
			sub, ok := child.(*property.SubProperty)
			if !ok {
				continue
			}
			if normalizeId(sub.Name()) != target {
				continue
			}
			cp := findSubCanvas(sub.Children(), "icon")
			if cp == nil {
				return nil, ErrNotFound
			}
			return decodeCanvas(f, cp)
		}
	}
	return nil, ErrNotFound
}

// extractEntityIcon resolves a single entity id from a flat WZ file
// (Npc.wz / Mob.wz / Reactor.wz) and returns the decoded canvas chosen by
// finder. Falls back to info/link redirection.
func extractEntityIcon(f *wz.File, id uint32, finder canvasFinder) (image.Image, error) {
	if f == nil {
		return nil, ErrNotFound
	}
	root := f.Root()
	if root == nil {
		return nil, ErrNotFound
	}

	imagesByName := make(map[string]*wz.Image)
	for _, img := range root.Images() {
		imagesByName[normalizeId(img.Name())] = img
	}

	target := strconv.FormatUint(uint64(id), 10)
	for _, img := range root.Images() {
		if normalizeId(img.Name()) != target {
			continue
		}
		props := img.Properties()
		if len(props) == 0 {
			return nil, ErrNotFound
		}
		cp := finder(props)
		if cp == nil {
			cp = resolveLinkedCanvas(imagesByName, props, finder)
		}
		if cp == nil {
			return nil, ErrNotFound
		}
		return decodeCanvas(f, cp)
	}
	return nil, ErrNotFound
}

// resolveLinkedCanvas follows info/link string properties to find a canvas
// from a linked entity. Follows up to 5 links to avoid infinite cycles.
func resolveLinkedCanvas(images map[string]*wz.Image, props []property.Property, finder canvasFinder) *property.CanvasProperty {
	for depth := 0; depth < 5; depth++ {
		linkId := findInfoLink(props)
		if linkId == "" {
			return nil
		}
		linked := findImageById(images, linkId)
		if linked == nil {
			return nil
		}
		linkedProps := linked.Properties()
		cp := finder(linkedProps)
		if cp != nil {
			return cp
		}
		props = linkedProps
	}
	return nil
}

// findInfoLink extracts the "link" string value from the "info" sub-property,
// if present.
func findInfoLink(props []property.Property) string {
	info := findSub(props, "info")
	if info == nil {
		return ""
	}
	for _, p := range info.Children() {
		if sp, ok := p.(*property.StringProperty); ok && sp.Name() == "link" {
			return sp.Value()
		}
	}
	return ""
}

// findImageById looks up an image by its numeric id. The `images` map is
// keyed by normalizeId(img.Name()), so any of the forms a UOL might use
// (raw id, zero-padded, with or without `.img`) reduces to the same key
// after normalization.
func findImageById(images map[string]*wz.Image, id string) *wz.Image {
	if img, ok := images[normalizeId(id)]; ok {
		return img
	}
	return nil
}

// indexItemSubs returns every top-level SubProperty in a multi-item image
// keyed by its raw name (zero-padded form, as the WZ UOL paths use).
func indexItemSubs(props []property.Property) map[string]*property.SubProperty {
	out := make(map[string]*property.SubProperty, len(props))
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok {
			out[sub.Name()] = sub
		}
	}
	return out
}

// findInfoIconUOL returns the UOL raw value stored under info/icon, if the
// icon is a UOL reference rather than a direct canvas.
func findInfoIconUOL(props []property.Property) string {
	info := findSub(props, "info")
	if info == nil {
		return ""
	}
	for _, c := range info.Children() {
		if uol, ok := c.(*property.UOLProperty); ok && uol.Name() == "icon" {
			return uol.Value()
		}
	}
	return ""
}

// resolveItemIconUOL resolves info/icon UOL references of the shape
// "../../<siblingId>/info/icon" against the enclosing multi-item image's
// top-level sub-properties. Chains are followed up to 5 hops with cycle
// detection.
func resolveItemIconUOL(siblings map[string]*property.SubProperty, fromName string, props []property.Property) *property.CanvasProperty {
	visited := map[string]struct{}{fromName: {}}
	for depth := 0; depth < 5; depth++ {
		uolPath := findInfoIconUOL(props)
		if uolPath == "" {
			return nil
		}
		targetName, tail, ok := splitSiblingUOL(uolPath)
		if !ok {
			return nil
		}
		if _, seen := visited[targetName]; seen {
			return nil
		}
		visited[targetName] = struct{}{}

		target, ok := siblings[targetName]
		if !ok {
			return nil
		}
		if tail == "info/icon" {
			if cp := findInfoIcon(target.Children()); cp != nil {
				return cp
			}
			props = target.Children()
			fromName = targetName
			continue
		}
		return nil
	}
	return nil
}

// splitSiblingUOL parses a UOL path of the form "../../<name>/<tail...>"
// and returns the sibling sub-property name and the remaining tail. Paths
// with a different leading dot-count (crossing images) are reported as
// unsupported.
func splitSiblingUOL(uol string) (string, string, bool) {
	parts := strings.Split(uol, "/")
	dots := 0
	for dots < len(parts) && parts[dots] == ".." {
		dots++
	}
	if dots != 2 || len(parts) < dots+2 {
		return "", "", false
	}
	name := parts[dots]
	tail := strings.Join(parts[dots+1:], "/")
	return name, tail, true
}

// findStandCanvas finds the stand/0 canvas for NPCs and mobs. Falls back to
// move/0 or any first canvas in any sub.
func findStandCanvas(props []property.Property) *property.CanvasProperty {
	if standDir := findSub(props, "stand"); standDir != nil {
		if cp := findFirstCanvas(standDir.Children()); cp != nil {
			return cp
		}
	}
	if moveDir := findSub(props, "move"); moveDir != nil {
		if cp := findFirstCanvas(moveDir.Children()); cp != nil {
			return cp
		}
	}
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok {
			if cp := findFirstCanvas(sub.Children()); cp != nil {
				return cp
			}
		}
	}
	return nil
}

// findReactorCanvas finds the 0/0 canvas for reactors.
func findReactorCanvas(props []property.Property) *property.CanvasProperty {
	zeroDir := findSub(props, "0")
	if zeroDir != nil {
		if cp := findFirstCanvas(zeroDir.Children()); cp != nil {
			return cp
		}
	}
	return nil
}

// findInfoIcon finds the info/icon canvas for items.
func findInfoIcon(props []property.Property) *property.CanvasProperty {
	info := findSub(props, "info")
	if info == nil {
		return nil
	}
	return findSubCanvas(info.Children(), "icon")
}

// findSub finds a named SubProperty in a property list.
func findSub(props []property.Property, name string) *property.SubProperty {
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok && sub.Name() == name {
			return sub
		}
	}
	return nil
}

// findSubCanvas finds a named CanvasProperty in a property list.
func findSubCanvas(props []property.Property, name string) *property.CanvasProperty {
	for _, p := range props {
		if cp, ok := p.(*property.CanvasProperty); ok && cp.Name() == name {
			return cp
		}
	}
	return nil
}

// findFirstCanvas finds the first CanvasProperty or the first canvas inside
// a "0" sub-property.
func findFirstCanvas(props []property.Property) *property.CanvasProperty {
	for _, p := range props {
		if cp, ok := p.(*property.CanvasProperty); ok {
			return cp
		}
	}
	zero := findSub(props, "0")
	if zero != nil {
		for _, p := range zero.Children() {
			if cp, ok := p.(*property.CanvasProperty); ok {
				return cp
			}
		}
	}
	return nil
}

// normalizeId strips a trailing ".img" suffix (present on top-level WZ image
// names like "0100100.img") and leading zeros from a WZ entity id. Callers
// pass raw image / sub-property names; this canonicalizes both forms so the
// comparison against a uint32-string target (e.g. "100100") matches.
func normalizeId(id string) string {
	id = strings.TrimSuffix(id, ".img")
	trimmed := strings.TrimLeft(id, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

// decodeCanvas reads canvas data from the WZ file and decompresses it into
// an in-memory image. No filesystem I/O.
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
