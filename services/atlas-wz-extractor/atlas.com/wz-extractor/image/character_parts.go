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
	"strings"

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
// "default" is included for equipment that doesn't animate (hair, face, hats,
// gloves, glasses, earrings, etc.) — those have direct canvas children rather
// than a frame SubProperty layer.
var stancesInScope = map[string]struct{}{
	"stand1":  {},
	"stand2":  {},
	"walk1":   {},
	"alert":   {},
	"jump":    {},
	"default": {},
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

// extractPartCanvas decodes a single part canvas, writes the PNG, and writes
// the JSON sidecar. The destination dir is created if missing.
func extractPartCanvas(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, dir, partName string) error {
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

// extractDefaultStanceChildren writes canvas parts from a `default` stance
// directly into {templateDir}/default/0/. Unlike animated stances, `default`
// has no frame sub-property layer — its children are CanvasProperties directly.
// This helper is factored out so it can be unit-tested without a real WZ file.
func extractDefaultStanceChildren(l logrus.FieldLogger, f *wz.File, templateId string, children []property.Property, templateDir string) int {
	frameDir := filepath.Join(templateDir, "default", "0")
	count := 0
	for _, partProp := range children {
		cp, ok := partProp.(*property.CanvasProperty)
		if !ok {
			continue
		}
		if err := extractPartCanvas(l, f, cp, frameDir, cp.Name()); err != nil {
			l.WithError(err).Warnf("extract part %s/default/0/%s", templateId, cp.Name())
			continue
		}
		count++
	}
	return count
}

// extractTemplateImg processes one Character.wz .img file. It writes
// {outRoot}/{templateId}/info.json plus, for every supported stance/frame
// canvas, {outRoot}/{templateId}/{stance}/{frame}/{part}.png + .json.
func extractTemplateImg(l logrus.FieldLogger, f *wz.File, img *wz.Image, outRoot string) (int, error) {
	templateId := normalizeId(img.Name())
	templateDir := filepath.Join(outRoot, templateId)

	info := extractInfoBlock(img.Properties())
	if err := writeInfoJSON(templateDir, info); err != nil {
		return 0, fmt.Errorf("write info: %w", err)
	}

	count := 0
	for _, p := range img.Properties() {
		stanceSub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		stance := stanceSub.Name()
		if _, ok := stancesInScope[stance]; !ok {
			continue
		}
		if stance == "default" {
			// `default` has no frame sub-property layer; its children are
			// CanvasProperties directly. Synthesise frame 0 on disk.
			count += extractDefaultStanceChildren(l, f, templateId, stanceSub.Children(), templateDir)
			continue
		}
		for _, fp := range stanceSub.Children() {
			frameSub, ok := fp.(*property.SubProperty)
			if !ok {
				continue
			}
			frameName := frameSub.Name()
			frameDir := filepath.Join(templateDir, stance, frameName)
			for _, partProp := range frameSub.Children() {
				cp, ok := partProp.(*property.CanvasProperty)
				if !ok {
					continue
				}
				if err := extractPartCanvas(l, f, cp, frameDir, cp.Name()); err != nil {
					l.WithError(err).Warnf("extract part %s/%s/%s/%s", templateId, stance, frameName, cp.Name())
					continue
				}
				count++
			}
		}
	}
	return count, nil
}

// extractCharacterParts walks Character.wz: every .img at the root (body
// skins) plus every .img in equipmentSubdirs, emitting per-template assets.
func extractCharacterParts(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}
	tenantOut := filepath.Join(outputDir, "character-parts")
	total := 0

	for _, img := range root.Images() {
		// Only body skin imgs live at the root; their names start with "0000" or "0001".
		if !strings.HasPrefix(img.Name(), "0000") && !strings.HasPrefix(img.Name(), "0001") {
			continue
		}
		n, err := extractTemplateImg(l, f, img, tenantOut)
		if err != nil {
			l.WithError(err).Warnf("extract body skin %s", img.Name())
			continue
		}
		total += n
	}
	for _, sub := range equipmentSubdirs {
		dir := findCharSubdir(root.Directories(), sub)
		if dir == nil {
			continue
		}
		for _, img := range dir.Images() {
			n, err := extractTemplateImg(l, f, img, tenantOut)
			if err != nil {
				l.WithError(err).Warnf("extract %s/%s", sub, img.Name())
				continue
			}
			total += n
		}
	}
	l.Infof("Extracted [%d] character part canvases.", total)
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
