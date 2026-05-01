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
var stancesInScope = map[string]struct{}{
	"stand1": {},
	"stand2": {},
	"walk1":  {},
	"alert":  {},
	"jump":   {},
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
