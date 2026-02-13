package image

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/canvas"
	"atlas-wz-extractor/wz/property"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// ExtractIcons extracts domain imagery (NPC, mob, item, skill, reactor, equipment icons) to the output directory.
// Output structure: {outputDir}/{category}/{id}/icon.png
func ExtractIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	name := strings.ToLower(f.Name())

	switch {
	case name == "npc":
		return extractEntityIcons(l, f, outputDir, "npc", findStandCanvas)
	case name == "mob":
		return extractEntityIcons(l, f, outputDir, "mob", findStandCanvas)
	case name == "reactor":
		return extractEntityIcons(l, f, outputDir, "reactor", findReactorCanvas)
	case name == "item":
		return extractItemIcons(l, f, outputDir)
	case name == "skill":
		return extractSkillIcons(l, f, outputDir)
	case name == "character":
		return extractEquipmentIcons(l, f, outputDir)
	default:
		return nil
	}
}

// canvasFinder is a function that finds the appropriate canvas from an image's properties.
type canvasFinder func(props []property.Property) *property.CanvasProperty

// extractEntityIcons extracts icons from a flat WZ file (e.g., Npc.wz, Mob.wz).
// Each image at the root level represents an entity by ID.
func extractEntityIcons(l logrus.FieldLogger, f *wz.File, outputDir, category string, finder canvasFinder) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	count := 0
	for _, img := range root.Images() {
		entityId := normalizeId(img.Name())
		props := img.Properties()
		if len(props) == 0 {
			continue
		}

		cp := finder(props)
		if cp == nil {
			continue
		}

		if err := writeCanvasPng(l, f, cp, outputDir, category, entityId); err != nil {
			l.WithError(err).Warnf("Unable to extract icon for %s [%s].", category, entityId)
			continue
		}
		count++
	}
	l.Infof("Extracted [%d] %s icons.", count, category)
	return nil
}

// extractItemIcons extracts item icons from Item.wz.
// Items are organized in subdirectories by category (Cash, Consume, Etc, Install, Pet).
// Some categories store one item per .img (e.g., Pet), while others store multiple items
// per .img as sub-properties (e.g., Cash, Consume, Etc, Install).
func extractItemIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	count := 0
	for _, dir := range root.Directories() {
		for _, img := range dir.Images() {
			props := img.Properties()
			if len(props) == 0 {
				continue
			}

			// Check if this image directly has info/icon (single-item images like Pet)
			cp := findInfoIcon(props)
			if cp != nil {
				if err := writeCanvasPng(l, f, cp, outputDir, "item", normalizeId(img.Name())); err != nil {
					l.WithError(err).Warnf("Unable to extract icon for item [%s].", img.Name())
				} else {
					count++
				}
				continue
			}

			// Multi-item image: each sub-property is an item with its own info/icon
			for _, p := range props {
				sub, ok := p.(*property.SubProperty)
				if !ok {
					continue
				}
				cp = findInfoIcon(sub.Children())
				if cp == nil {
					continue
				}
				if err := writeCanvasPng(l, f, cp, outputDir, "item", normalizeId(sub.Name())); err != nil {
					l.WithError(err).Warnf("Unable to extract icon for item [%s].", sub.Name())
					continue
				}
				count++
			}
		}
	}
	l.Infof("Extracted [%d] item icons.", count)
	return nil
}

// extractSkillIcons extracts skill icons from Skill.wz.
// Each image represents a skill book, with individual skills as sub-properties.
func extractSkillIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	count := 0
	for _, img := range root.Images() {
		props := img.Properties()
		if len(props) == 0 {
			continue
		}

		// Look for "skill" sub-directory containing individual skills
		skillDir := findSub(props, "skill")
		if skillDir == nil {
			continue
		}

		for _, child := range skillDir.Children() {
			sub, ok := child.(*property.SubProperty)
			if !ok {
				continue
			}
			skillId := normalizeId(sub.Name())
			cp := findSubCanvas(sub.Children(), "icon")
			if cp == nil {
				continue
			}

			if err := writeCanvasPng(l, f, cp, outputDir, "skill", skillId); err != nil {
				l.WithError(err).Warnf("Unable to extract icon for skill [%s].", skillId)
				continue
			}
			count++
		}
	}
	l.Infof("Extracted [%d] skill icons.", count)
	return nil
}

// findStandCanvas finds the stand/0 canvas for NPCs and mobs.
// Falls back to info/link if present (linked entity), or any first canvas found.
func findStandCanvas(props []property.Property) *property.CanvasProperty {
	// Try stand/0
	standDir := findSub(props, "stand")
	if standDir != nil {
		cp := findFirstCanvas(standDir.Children())
		if cp != nil {
			return cp
		}
	}

	// Try move/0
	moveDir := findSub(props, "move")
	if moveDir != nil {
		cp := findFirstCanvas(moveDir.Children())
		if cp != nil {
			return cp
		}
	}

	// Try any first canvas in any sub
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok {
			cp := findFirstCanvas(sub.Children())
			if cp != nil {
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
		cp := findFirstCanvas(zeroDir.Children())
		if cp != nil {
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

// findFirstCanvas finds the first CanvasProperty or the first canvas inside a "0" sub-property.
func findFirstCanvas(props []property.Property) *property.CanvasProperty {
	// Direct canvas
	for _, p := range props {
		if cp, ok := p.(*property.CanvasProperty); ok {
			return cp
		}
	}
	// Check "0" sub
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

// extractEquipmentIcons extracts equipment icons from Character.wz.
// Equipment items are organized in subdirectories by type (Weapon, Cap, Coat, etc.).
// Each .img file represents a single equipment item with an info/icon canvas.
func extractEquipmentIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	count := 0
	for _, dir := range root.Directories() {
		for _, img := range dir.Images() {
			props := img.Properties()
			if len(props) == 0 {
				continue
			}

			cp := findInfoIcon(props)
			if cp == nil {
				continue
			}

			if err := writeCanvasPng(l, f, cp, outputDir, "item", normalizeId(img.Name())); err != nil {
				l.WithError(err).Warnf("Unable to extract icon for equipment [%s].", img.Name())
				continue
			}
			count++
		}
	}
	l.Infof("Extracted [%d] equipment icons.", count)
	return nil
}

// normalizeId strips leading zeros from a WZ entity ID to produce a numeric string
// that matches how the web UI formats entity IDs (as numbers without zero-padding).
func normalizeId(id string) string {
	trimmed := strings.TrimLeft(id, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

var canvasDiagCount int32

// writeCanvasPng reads canvas data from the WZ file, decompresses it, and writes a PNG.
func writeCanvasPng(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, outputDir, category, entityId string) error {
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return fmt.Errorf("unable to read canvas data: %w", err)
	}

	canvasDiagCount++
	if canvasDiagCount <= 10 {
		headerBytes := data
		if len(headerBytes) > 16 {
			headerBytes = headerBytes[:16]
		}
		l.Infof("[DIAG] Canvas %s/%s: format=%d, size=%dx%d, dataLen=%d, dataOffset=%d, first16bytes=%X",
			category, entityId, cp.Format(), cp.Width(), cp.Height(), len(data), cp.DataOffset(), headerBytes)
	}

	img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		if canvasDiagCount <= 10 {
			l.WithError(err).Errorf("[DIAG] Decompression FAILED for %s/%s", category, entityId)
		}
		return fmt.Errorf("unable to decompress canvas: %w", err)
	}

	dir := filepath.Join(outputDir, category, entityId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create directory [%s]: %w", dir, err)
	}

	outPath := filepath.Join(dir, "icon.png")
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("unable to create PNG file [%s]: %w", outPath, err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("unable to encode PNG: %w", err)
	}

	return nil
}
