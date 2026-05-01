package image

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// extractCharacterMaps reads zmap.img and smap.img from a Base.wz file and
// writes character-meta/zmap.json and character-meta/smap.json so the render
// service can resolve sprite z-order and slot-precedence at composition time.
func extractCharacterMaps(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	dir := filepath.Join(outputDir, "character-meta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create character-meta dir: %w", err)
	}

	if err := writeZmap(l, root.Images(), dir); err != nil {
		l.WithError(err).Warn("zmap extraction failed")
	}
	if err := writeSmap(l, root.Images(), dir); err != nil {
		l.WithError(err).Warn("smap extraction failed")
	}
	return nil
}

// writeZmap serializes Base.wz/zmap.img as an ordered list of layer-string
// names. The order in the WZ is the render order.
func writeZmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	zmap := findImage(images, "zmap")
	if zmap == nil {
		return fmt.Errorf("zmap.img not found in Base.wz")
	}
	return writeZmapFromProps(zmap.Properties(), dir)
}

func writeZmapFromProps(props []property.Property, dir string) error {
	out := make([]string, 0, len(props))
	for _, p := range props {
		out = append(out, p.Name())
	}
	return writeJSON(filepath.Join(dir, "zmap.json"), out)
}

// writeSmap serializes Base.wz/smap.img as a layer-string -> slot-categories
// map (the WZ value is a string of slot codes, e.g. "CpHdH1H2H3...").
func writeSmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	smap := findImage(images, "smap")
	if smap == nil {
		return fmt.Errorf("smap.img not found in Base.wz")
	}
	return writeSmapFromProps(smap.Properties(), dir)
}

func writeSmapFromProps(props []property.Property, dir string) error {
	out := map[string]string{}
	for _, p := range props {
		if sp, ok := p.(*property.StringProperty); ok {
			out[sp.Name()] = sp.Value()
		}
	}
	return writeJSON(filepath.Join(dir, "smap.json"), out)
}

func findImage(images []*wz.Image, name string) *wz.Image {
	for _, img := range images {
		if strings.EqualFold(img.Name(), name) {
			return img
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
