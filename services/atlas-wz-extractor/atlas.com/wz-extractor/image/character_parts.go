package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
