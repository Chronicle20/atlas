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

// ExtractMinimaps extracts miniMap/canvas from every map image inside Map.wz into
// {outputDir}/map/{mapId}/minimap.png. Maps without a miniMap canvas are skipped.
func ExtractMinimaps(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	if strings.ToLower(f.Name()) != "map" {
		return nil
	}
	root := f.Root()
	if root == nil {
		return nil
	}

	mapDir := findDirectory(root.Directories(), "Map")
	if mapDir == nil {
		l.Debug("Map.wz has no /Map directory; skipping minimap extraction.")
		return nil
	}

	extracted := 0
	skipped := 0
	for _, sub := range mapDir.Directories() {
		for _, img := range sub.Images() {
			props := img.Properties()
			if len(props) == 0 {
				continue
			}
			cp := findMinimapCanvas(props)
			if cp == nil {
				skipped++
				continue
			}
			if err := writeMinimapPng(l, f, cp, outputDir, normalizeId(img.Name())); err != nil {
				l.WithError(err).Warnf("Unable to extract minimap for map [%s].", img.Name())
				continue
			}
			extracted++
		}
	}
	l.Infof("Extracted [%d] map minimaps (skipped [%d] without minimap).", extracted, skipped)
	return nil
}

// findMinimapCanvas returns the miniMap/canvas property, or nil if absent.
func findMinimapCanvas(props []property.Property) *property.CanvasProperty {
	mm := findSub(props, "miniMap")
	if mm == nil {
		return nil
	}
	return findSubCanvas(mm.Children(), "canvas")
}

// findDirectory returns the named child directory, or nil.
func findDirectory(dirs []*wz.Directory, name string) *wz.Directory {
	for _, d := range dirs {
		if d.Name() == name {
			return d
		}
	}
	return nil
}

// writeMinimapPng decompresses the canvas and writes it as {outputDir}/map/{mapId}/minimap.png.
func writeMinimapPng(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, outputDir, mapId string) error {
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return fmt.Errorf("read canvas data: %w", err)
	}
	img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		return fmt.Errorf("decompress canvas: %w", err)
	}

	dir := filepath.Join(outputDir, "map", mapId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir [%s]: %w", dir, err)
	}
	outPath := filepath.Join(dir, "minimap.png")
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create PNG [%s]: %w", outPath, err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}
	_ = l
	return nil
}
