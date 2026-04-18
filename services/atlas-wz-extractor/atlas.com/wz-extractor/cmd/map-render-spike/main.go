// Dev harness for the mapimage package. Opens a Map.wz, resolves a map id, and
// writes render.png to --out. Promoted library lives in atlas-wz-extractor/mapimage.
package main

import (
	"atlas-wz-extractor/mapimage"
	"atlas-wz-extractor/wz"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	wzPath := flag.String("wz", "", "path to Map.wz")
	mapId := flag.String("map", "100000000", "map id")
	out := flag.String("out", "/tmp/render.png", "output PNG path")
	flag.Parse()
	if *wzPath == "" {
		log.Fatal("--wz required")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	if os.Getenv("DEBUG") != "" {
		logger.SetLevel(logrus.DebugLevel)
	}

	f, err := wz.Open(logger, *wzPath)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer f.Close()

	idx := mapimage.NewIndex(f)
	maps := idx.Maps()
	img, ok := maps[*mapId]
	if !ok {
		padded := fmt.Sprintf("%09s", *mapId)
		img, ok = maps[padded]
	}
	if !ok {
		log.Fatalf("map %s not found", *mapId)
	}

	// Render writes to {outDir}/map/{mapId}/render.png.
	outDir, err := os.MkdirTemp("", "mapimage-spike-")
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	stats, err := mapimage.Render(context.Background(), logger, idx, img, outDir, mapimage.Options{RenderBackgrounds: true})
	if err != nil {
		log.Fatalf("render: %v", err)
	}

	if err := os.Rename(stats.Output, *out); err != nil {
		// Fallback to copy if rename crosses devices.
		if err := copyFile(stats.Output, *out); err != nil {
			log.Fatalf("move output: %v", err)
		}
	}
	logger.Infof("wrote %s (%dx%d, %d sprites, %dms)", *out, stats.Width, stats.Height, stats.SpriteCount, stats.DurationMs)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	outF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outF.Close()
	_, err = io.Copy(outF, in)
	return err
}
