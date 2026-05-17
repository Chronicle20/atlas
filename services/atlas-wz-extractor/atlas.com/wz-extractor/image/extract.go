package image

import (
	"atlas-wz-extractor/extraction/parallelism"
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/canvas"
	"atlas-wz-extractor/wz/property"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// ExtractIcons extracts domain imagery (NPC, mob, item, skill, reactor, equipment icons,
// UI world icons) to the output directory.
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
		if err := extractEquipmentIcons(l, f, outputDir); err != nil {
			l.WithError(err).Warn("equipment icons extraction failed")
		}
		return extractCharacterParts(l, f, outputDir)
	case name == "ui":
		return extractUIIcons(l, f, outputDir)
	case name == "base":
		return extractCharacterMaps(l, f, outputDir)
	default:
		return nil
	}
}

// extractUIIcons extracts world icons from UI.wz/Login.img/ViewAllChar/WorldIcons.
// Each canvas under WorldIcons is keyed by world id (e.g. "0", "1", … "20") and is
// the small (~20×20) icon shown next to a world name in the world list.
// Output: {outputDir}/world-icon/{worldId}/icon.png
func extractUIIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	var loginProps []property.Property
	for _, img := range root.Images() {
		if strings.EqualFold(img.Name(), "Login") {
			loginProps = img.Properties()
			break
		}
	}
	if loginProps == nil {
		l.Debugf("UI.wz has no Login.img — skipping world icon extraction.")
		return nil
	}

	canvases := findWorldIconCanvases(loginProps)
	if len(canvases) == 0 {
		l.Debugf("UI.wz Login.img/ViewAllChar/WorldIcons missing or empty — skipping world icon extraction.")
		return nil
	}

	count := 0
	for worldId, cp := range canvases {
		if err := writeCanvasPng(l, f, cp, outputDir, "world-icon", worldId); err != nil {
			l.WithError(err).Warnf("Unable to extract world icon [%s].", worldId)
			continue
		}
		count++
	}
	l.Infof("Extracted [%d] world icons.", count)
	return nil
}

// findWorldIconCanvases walks the Login.img property tree to find every
// ViewAllChar/WorldIcons canvas, keyed by normalized world id. Exposed for
// unit testing without a real WZ file backing.
func findWorldIconCanvases(loginProps []property.Property) map[string]*property.CanvasProperty {
	viewAllChar := findSub(loginProps, "ViewAllChar")
	if viewAllChar == nil {
		return nil
	}
	worldIcons := findSub(viewAllChar.Children(), "WorldIcons")
	if worldIcons == nil {
		return nil
	}
	out := make(map[string]*property.CanvasProperty)
	for _, child := range worldIcons.Children() {
		cp, ok := child.(*property.CanvasProperty)
		if !ok {
			continue
		}
		out[normalizeId(cp.Name())] = cp
	}
	return out
}

// canvasFinder is a function that finds the appropriate canvas from an image's properties.
type canvasFinder func(props []property.Property) *property.CanvasProperty

// extractEntityIcons extracts icons from a flat WZ file (e.g., Npc.wz, Mob.wz).
// Each image at the root level represents an entity by ID.
// When an entity has no canvas but has an info/link string property pointing to another
// entity ID, the linked entity's canvas is used instead.
func extractEntityIcons(l logrus.FieldLogger, f *wz.File, outputDir, category string, finder canvasFinder) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	// Build a lookup map for resolving inter-entity links (e.g., mob 9300145 -> mob 6110300).
	imagesByName := make(map[string]*wz.Image)
	for _, img := range root.Images() {
		imagesByName[img.Name()] = img
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
			cp = resolveLinkedCanvas(l, imagesByName, props, finder)
		}
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

// resolveLinkedCanvas follows info/link string properties to find a canvas from a linked entity.
// Follows up to 5 links to avoid infinite cycles.
func resolveLinkedCanvas(l logrus.FieldLogger, images map[string]*wz.Image, props []property.Property, finder canvasFinder) *property.CanvasProperty {
	for depth := 0; depth < 5; depth++ {
		linkId := findInfoLink(props)
		if linkId == "" {
			return nil
		}

		linked := findImageById(images, linkId)
		if linked == nil {
			l.Debugf("Linked entity [%s] not found in WZ file.", linkId)
			return nil
		}

		linkedProps := linked.Properties()
		cp := finder(linkedProps)
		if cp != nil {
			return cp
		}

		// The linked entity may itself be a link — continue resolving.
		props = linkedProps
	}
	return nil
}

// findInfoLink extracts the "link" string value from the "info" sub-property, if present.
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

// findImageById looks up an image by its numeric ID, padding with leading zeros as needed.
// WZ image names are zero-padded to 7 digits (e.g., "6110300" for mob 6110300).
func findImageById(images map[string]*wz.Image, id string) *wz.Image {
	if img, ok := images[id]; ok {
		return img
	}
	// Try zero-padded form (most mob IDs are 7-digit padded).
	padded := fmt.Sprintf("%07s", id)
	if img, ok := images[padded]; ok {
		return img
	}
	return nil
}

// extractItemIcons extracts item icons from Item.wz.
// Items are organized in subdirectories by category (Cash, Consume, Etc, Install, Pet).
// Some categories store one item per .img (e.g., Pet), while others store multiple items
// per .img as sub-properties (e.g., Cash, Consume, Etc, Install).
//
// Many items store info/icon as a UOLProperty — a relative WZ path such as
// "../../02040001/info/icon" pointing at another sibling item's canvas. These get
// resolved against the enclosing multi-item image so the linked canvas is extracted
// under the referencing item's id.
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

			// Single-item image (e.g., Pet): info/icon at the root.
			if cp := findInfoIcon(props); cp != nil {
				if err := writeCanvasPng(l, f, cp, outputDir, "item", normalizeId(img.Name())); err != nil {
					l.WithError(err).Warnf("Unable to extract icon for item [%s].", img.Name())
				} else {
					count++
				}
				continue
			}

			// Multi-item image: index siblings once per .img so UOL references resolve in O(1).
			siblings := indexItemSubs(props)

			for _, p := range props {
				sub, ok := p.(*property.SubProperty)
				if !ok {
					continue
				}
				cp := findInfoIcon(sub.Children())
				if cp == nil {
					cp = resolveItemIconUOL(l, siblings, sub.Name(), sub.Children())
				}
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

// indexItemSubs returns every top-level SubProperty in a multi-item image keyed by
// its raw name (zero-padded form, as the WZ UOL paths use).
func indexItemSubs(props []property.Property) map[string]*property.SubProperty {
	out := make(map[string]*property.SubProperty, len(props))
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok {
			out[sub.Name()] = sub
		}
	}
	return out
}

// findInfoIconUOL returns the UOL raw value stored under info/icon, if the icon is a
// UOL reference rather than a direct canvas.
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
// "../../<siblingId>/info/icon" against the enclosing multi-item image's top-level
// sub-properties. Chains are followed up to 5 hops with cycle detection.
func resolveItemIconUOL(l logrus.FieldLogger, siblings map[string]*property.SubProperty, fromName string, props []property.Property) *property.CanvasProperty {
	visited := map[string]struct{}{fromName: {}}
	for depth := 0; depth < 5; depth++ {
		uolPath := findInfoIconUOL(props)
		if uolPath == "" {
			return nil
		}
		targetName, tail, ok := splitSiblingUOL(uolPath)
		if !ok {
			l.Debugf("Unsupported UOL path shape [%s] from item [%s].", uolPath, fromName)
			return nil
		}
		if _, seen := visited[targetName]; seen {
			return nil
		}
		visited[targetName] = struct{}{}

		target, ok := siblings[targetName]
		if !ok {
			l.Debugf("UOL target [%s] not present among siblings (from item [%s]).", targetName, fromName)
			return nil
		}
		if tail == "info/icon" {
			if cp := findInfoIcon(target.Children()); cp != nil {
				return cp
			}
			// Target is itself a UOL — follow the chain.
			props = target.Children()
			fromName = targetName
			continue
		}
		l.Debugf("Unsupported UOL tail [%s] from item [%s].", tail, fromName)
		return nil
	}
	return nil
}

// splitSiblingUOL parses a UOL path of the form "../../<name>/<tail...>" and returns
// the sibling sub-property name and the remaining tail. Paths with a different leading
// dot-count (crossing images) are reported as unsupported.
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

// equipIconJob is a single unit of work for the equipment-icon worker pool.
type equipIconJob struct {
	imgName string
	itemId  string
	cp      *property.CanvasProperty
}

// extractEquipmentIcons extracts equipment icons from Character.wz.
// Equipment items are organized in subdirectories by type (Weapon, Cap, Coat, etc.).
// Each .img file represents a single equipment item with an info/icon canvas.
//
// Images are pre-parsed serially (because image parsing uses the shared
// seek-based wz.Reader), then dispatched to a worker pool sized from
// WZ_EXTRACT_PARALLELISM (see extraction/parallelism) for the CPU- and
// I/O-intensive canvas decode / PNG write phase. Per-job errors are
// logged but do not abort the pool, preserving the existing continue-
// on-error semantics.
func extractEquipmentIcons(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	itemOut := filepath.Join(outputDir, "item")

	// Pre-parse all images serially so workers only access already-cached data.
	var jobs []equipIconJob
	for _, dir := range root.Directories() {
		for _, img := range dir.Images() {
			props := img.Properties() // serial pre-parse
			if len(props) == 0 {
				continue
			}
			cp := findInfoIcon(props)
			if cp == nil {
				continue
			}
			jobs = append(jobs, equipIconJob{
				imgName: img.Name(),
				itemId:  normalizeId(img.Name()),
				cp:      cp,
			})
		}
	}

	// Worker pool sized from WZ_EXTRACT_PARALLELISM (env var).
	workers := parallelism.FromEnv(l)

	jobCh := make(chan equipIconJob, workers*2)
	var wg sync.WaitGroup
	var count atomic.Int64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				dir := filepath.Join(itemOut, j.itemId)
				if err := canvasWriter(l, f, j.cp, dir, "icon"); err != nil {
					l.WithError(err).Warnf("Unable to extract icon for equipment [%s].", j.imgName)
					continue
				}
				count.Add(1)
			}
		}()
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()

	l.Infof("Extracted [%d] equipment icons.", count.Load())
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
