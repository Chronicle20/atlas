package image

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/canvas"
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// partSidecar is the JSON sidecar emitted next to each part PNG.
type partSidecar struct {
	Origin vec            `json:"origin"`
	Map    map[string]vec `json:"map,omitempty"`
	Z      string         `json:"z,omitempty"`
	Group  string         `json:"group,omitempty"`
	Delay  int            `json:"delay,omitempty"`
	Face   int            `json:"face,omitempty"`
}

type vec struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// templateInfo is the per-img info.json block.
type templateInfo struct {
	Islot string `json:"islot,omitempty"`
	Vslot string `json:"vslot,omitempty"`
	Cash  int    `json:"cash"`
}

// stancesInScope is the explicit allow-list of stances we extract. Skipping
// fly/prone/swing/etc. keeps the on-disk footprint manageable.
//
// "default" is included for equipment that doesn't animate (hair, face, hats,
// gloves, glasses, earrings, etc.) — those have direct canvas children rather
// than a frame SubProperty layer.
//
// "front" and "back" are included because the head template (0001{wzSkin}.img)
// stores its head canvas under front/head and back/head — NOT under any of the
// animated stances. Stance dirs (stand1, stand2, walk1, etc.) only contain
// UOL aliases that point back to front/head.
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

// directCanvasStances are stances whose children are CanvasProperty parts at
// the top level (no frame SubProperty layer). Animated stances (stand1, etc.)
// instead nest frame SubProperties under the stance.
var directCanvasStances = map[string]struct{}{
	"default": {},
	"front":   {},
	"back":    {},
}

// equipmentSubdirs are the Character.wz subdirectories whose .img files we
// extract worn sprites for. Body skin imgs live at the root, not in a subdir.
var equipmentSubdirs = []string{
	"Cap", "Coat", "Longcoat", "Pants", "Shoes", "Glove",
	"Cape", "Shield", "Weapon", "Hair", "Face", "Accessory",
}

// extractInfoBlock returns a templateInfo populated from the `info` sub of
// an equipment img. Missing fields default to zero values.
func extractInfoBlock(props []property.Property) templateInfo {
	info := findSub(props, "info")
	if info == nil {
		return templateInfo{}
	}
	out := templateInfo{}
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

// writeInfoJSON writes {dir}/info.json.
func writeInfoJSON(dir string, info templateInfo) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "info.json"), b, 0o644)
}

// canvasWriter is the function used to emit a single part canvas to disk.
// It's a package-level variable so tests can stub it without a real *wz.File.
var canvasWriter = defaultCanvasWriter

// defaultCanvasWriter decodes a CanvasProperty against a real WZ file and
// writes the PNG + sidecar.
func defaultCanvasWriter(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, dir, partName string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return fmt.Errorf("read canvas: %w", err)
	}
	img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		return fmt.Errorf("decompress canvas: %w", err)
	}

	pngPath := filepath.Join(dir, partName+".png")
	out, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("create png: %w", err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}

	sidecar := buildPartSidecar(cp.Children())
	b, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, partName+".json"), b, 0o644)
}

// extractPartCanvas dispatches to the active canvasWriter (overridable in tests).
func extractPartCanvas(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, dir, partName string) error {
	return canvasWriter(l, f, cp, dir, partName)
}

// buildPartSidecar walks the children of a part canvas to produce the
// metadata sidecar. Children that are absent in the WZ stay zero-valued.
func buildPartSidecar(children []property.Property) partSidecar {
	out := partSidecar{Map: map[string]vec{}}
	for _, c := range children {
		switch v := c.(type) {
		case *property.VectorProperty:
			if v.Name() == "origin" {
				out.Origin = vec{X: int(v.X()), Y: int(v.Y())}
			}
		case *property.StringProperty:
			switch v.Name() {
			case "z":
				out.Z = v.Value()
			case "group":
				out.Group = v.Value()
			}
		case *property.IntProperty:
			if v.Name() == "delay" {
				out.Delay = int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == "face" {
				out.Face = int(v.Value())
			}
		case *property.SubProperty:
			if v.Name() == "map" {
				for _, jp := range v.Children() {
					if jv, ok := jp.(*property.VectorProperty); ok {
						out.Map[jv.Name()] = vec{X: int(jv.X()), Y: int(jv.Y())}
					}
				}
			}
		}
	}
	if len(out.Map) == 0 {
		out.Map = nil
	}
	return out
}

// pathLookup is a lower-cased path -> property map built once per .img to
// resolve UOL references in O(1).
type pathLookup map[string]property.Property

// buildPathLookup walks every property under root (recursively, including
// canvas children) and indexes them by their slash-joined absolute path. The
// root itself is not included; its top-level children appear as "name".
//
// Names are lower-cased so case-insensitive lookups (matching MapleStory's
// WZ behavior) work uniformly.
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
// slash-joined absolute path of the property containing the UOL — e.g.
// "stand1/0" for a UOLProperty at stand1/0/head). The result is the
// slash-joined absolute path of the UOL target.
//
// `..` segments pop one component; named segments push.
func canonicalizeUOLPath(anchorPath, uolValue string) string {
	parts := strings.Split(anchorPath, "/")
	if anchorPath == "" {
		parts = nil
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

// resolveUOL dereferences a UOL property given the anchor path of its
// containing SubProperty. The lookup is case-insensitive (paths in the
// pathLookup map are lower-cased). Chains of UOLs are followed up to 5 hops.
//
// Returns nil if the path doesn't resolve or if the chain exceeds 5 hops.
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
		// Anchor for the next hop is the parent path of the resolved UOL.
		currentAnchor = parentPath(target)
		current = next
	}
	return nil
}

// parentPath returns the slash-joined path with the last segment removed.
func parentPath(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// extractDefaultStanceChildren writes canvas parts from a stance whose
// children are CanvasProperties directly (default, front, back) into
// {templateDir}/{stance}/0/. Resolves UOL children against the .img-wide
// pathLookup so aliases are materialized.
//
// `lookup` and `stancePath` may be nil/"" if no UOL children are expected.
func extractDefaultStanceChildren(l logrus.FieldLogger, f *wz.File, templateId string, children []property.Property, templateDir, stance string, lookup pathLookup, stancePath string) int {
	frameDir := filepath.Join(templateDir, stance, "0")
	count := 0
	for _, partProp := range children {
		switch v := partProp.(type) {
		case *property.CanvasProperty:
			if err := extractPartCanvas(l, f, v, frameDir, v.Name()); err != nil {
				l.WithError(err).Warnf("extract part %s/%s/0/%s", templateId, stance, v.Name())
				continue
			}
			count++
		case *property.UOLProperty:
			if lookup == nil {
				continue
			}
			target := resolveUOL(lookup, stancePath, v)
			cp, ok := target.(*property.CanvasProperty)
			if !ok {
				continue
			}
			if err := extractPartCanvas(l, f, cp, frameDir, v.Name()); err != nil {
				l.WithError(err).Warnf("extract uol part %s/%s/0/%s", templateId, stance, v.Name())
				continue
			}
			count++
		}
	}
	return count
}

// extractAnimatedFrameChildren writes canvas parts from one frame of an
// animated stance (e.g., stand1/0/...), resolving UOL aliases along the way.
func extractAnimatedFrameChildren(l logrus.FieldLogger, f *wz.File, templateId, stance, frameName string, frameProps []property.Property, templateDir string, lookup pathLookup, framePath string) int {
	frameDir := filepath.Join(templateDir, stance, frameName)
	count := 0
	for _, partProp := range frameProps {
		switch v := partProp.(type) {
		case *property.CanvasProperty:
			if err := extractPartCanvas(l, f, v, frameDir, v.Name()); err != nil {
				l.WithError(err).Warnf("extract part %s/%s/%s/%s", templateId, stance, frameName, v.Name())
				continue
			}
			count++
		case *property.UOLProperty:
			target := resolveUOL(lookup, framePath, v)
			cp, ok := target.(*property.CanvasProperty)
			if !ok {
				continue
			}
			if err := extractPartCanvas(l, f, cp, frameDir, v.Name()); err != nil {
				l.WithError(err).Warnf("extract uol part %s/%s/%s/%s", templateId, stance, frameName, v.Name())
				continue
			}
			count++
		}
	}
	return count
}

// extractTemplateImg processes one Character.wz .img file. It writes
// {outRoot}/{templateId}/info.json plus, for every supported stance/frame
// canvas, {outRoot}/{templateId}/{stance}/{frame}/{part}.png + .json.
//
// UOL aliases under any in-scope stance are dereferenced and the target
// canvas is materialized at the alias's path so consumers don't need to
// understand the WZ link semantics.
func extractTemplateImg(l logrus.FieldLogger, f *wz.File, img *wz.Image, outRoot string) (int, error) {
	templateId := normalizeId(img.Name())
	templateDir := filepath.Join(outRoot, templateId)

	props := img.Properties()
	info := extractInfoBlock(props)
	if err := writeInfoJSON(templateDir, info); err != nil {
		return 0, fmt.Errorf("write info: %w", err)
	}

	lookup := buildPathLookup(props)

	count := 0
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
			count += extractDefaultStanceChildren(l, f, templateId, stanceSub.Children(), templateDir, stance, lookup, stancePath)
			continue
		}
		// Animated stance: each child is a frame SubProperty.
		for _, fp := range stanceSub.Children() {
			frameSub, ok := fp.(*property.SubProperty)
			if !ok {
				continue
			}
			frameName := frameSub.Name()
			framePath := stancePath + "/" + strings.ToLower(frameName)
			count += extractAnimatedFrameChildren(l, f, templateId, stance, frameName, frameSub.Children(), templateDir, lookup, framePath)
		}
	}
	return count, nil
}

// charPartJob is a single unit of work for the character-parts worker pool.
type charPartJob struct {
	sub string    // subdomain label: e.g. "Cap", "Hair", or "(body)"
	img *wz.Image // image to extract
}

// extractCharacterParts walks Character.wz: every .img at the root (body
// skins) plus every .img in equipmentSubdirs, emitting per-template assets.
//
// Images are pre-parsed serially (because image parsing uses the shared
// seek-based wz.Reader), then dispatched to a runtime.NumCPU() worker pool
// for the CPU- and I/O-intensive canvas decode / PNG write phase.
// Per-job errors are logged but do not abort the pool, preserving the
// existing continue-on-error semantics.
func extractCharacterParts(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}
	tenantOut := filepath.Join(outputDir, "character-parts")

	// Collect all jobs; calling img.Properties() here parses each image
	// serially so that workers only access already-cached data (no seeks).
	var jobs []charPartJob

	for _, img := range root.Images() {
		if !strings.HasPrefix(img.Name(), "0000") && !strings.HasPrefix(img.Name(), "0001") {
			continue
		}
		img.Properties() // pre-parse before concurrent dispatch
		jobs = append(jobs, charPartJob{sub: "(body)", img: img})
	}

	subImgCounts := make(map[string]int, len(equipmentSubdirs))
	for _, sub := range equipmentSubdirs {
		dir := findCharSubdir(root.Directories(), sub)
		if dir == nil {
			continue
		}
		imgs := dir.Images()
		subImgCounts[sub] = len(imgs)
		for _, img := range imgs {
			img.Properties() // pre-parse before concurrent dispatch
			jobs = append(jobs, charPartJob{sub: sub, img: img})
		}
	}

	// Worker pool: runtime.NumCPU() goroutines consuming from jobCh.
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	jobCh := make(chan charPartJob, workers*2)
	var wg sync.WaitGroup
	var totalCount atomic.Int64
	var subTotals sync.Map // sub -> *atomic.Int64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			wl := l.WithField("worker", workerId)
			for j := range jobCh {
				n, err := extractTemplateImg(wl, f, j.img, tenantOut)
				if err != nil {
					wl.WithError(err).Warnf("extract %s/%s", j.sub, j.img.Name())
					continue
				}
				totalCount.Add(int64(n))
				cnt, _ := subTotals.LoadOrStore(j.sub, &atomic.Int64{})
				cnt.(*atomic.Int64).Add(int64(n))
			}
		}(i)
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()

	// Emit per-subdomain log lines matching the pre-existing format so
	// operators tracking progress see familiar markers.
	for _, sub := range equipmentSubdirs {
		v, ok := subTotals.Load(sub)
		if !ok {
			continue
		}
		l.Infof("Character.wz/%s: %d imgs, %d canvases extracted (running total %d).",
			sub, subImgCounts[sub], v.(*atomic.Int64).Load(), totalCount.Load())
	}
	if v, ok := subTotals.Load("(body)"); ok {
		l.Infof("Character.wz/(body): %d canvases extracted.", v.(*atomic.Int64).Load())
	}
	l.Infof("Extracted [%d] character part canvases.", totalCount.Load())
	return nil
}

func findCharSubdir(dirs []*wz.Directory, name string) *wz.Directory {
	for _, d := range dirs {
		if strings.EqualFold(d.Name(), name) {
			return d
		}
	}
	return nil
}
