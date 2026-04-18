package extraction

import (
	"atlas-wz-extractor/mapimage"
	"atlas-wz-extractor/wz"
	"context"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// RenderMaps iterates every map image in Map.wz and invokes mapimage.Render
// under a worker pool (runtime.NumCPU()). Empty and too-large maps are skipped
// with a debug-level log; renderer errors are logged as warnings and the batch
// continues.
//
// MaxPixels defaults to mapimage.DefaultMaxPixels; override via
// WZ_EXTRACT_MAX_MAP_PIXELS=<int>.
func RenderMaps(ctx context.Context, l logrus.FieldLogger, f *wz.File, outputDir string) error {
	if strings.ToLower(f.Name()) != "map" {
		return nil
	}

	idx := mapimage.NewIndex(f)
	maps := idx.Maps()
	if len(maps) == 0 {
		l.Debug("Map.wz has no /Map directory; skipping map render.")
		return nil
	}

	// Backgrounds are intentionally skipped — the scope decision from the
	// spike keeps the render black where no tile/obj was placed, avoiding
	// horizon-seam artifacts from the parallax collapse.
	opts := mapimage.Options{
		MaxPixels:         maxPixelsFromEnv(l),
		RenderBackgrounds: false,
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan *wz.Image)
	var wg sync.WaitGroup
	var rendered, skipped, failed uint64
	var counterMu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			workerLog := l.WithField("worker", workerId)
			for img := range jobs {
				if ctx.Err() != nil {
					return
				}
				stats, err := mapimage.Render(ctx, workerLog, idx, img, outputDir, opts)
				counterMu.Lock()
				switch {
				case errors.Is(err, mapimage.ErrSkipEmpty):
					skipped++
					counterMu.Unlock()
					workerLog.WithField("mapId", stats.MapID).Debug("map skipped: empty")
				case errors.Is(err, mapimage.ErrSkipTooLarge):
					skipped++
					counterMu.Unlock()
					workerLog.WithFields(logrus.Fields{
						"mapId":  stats.MapID,
						"width":  stats.Width,
						"height": stats.Height,
					}).Warn("map skipped: exceeds MaxPixels")
				case err != nil:
					failed++
					counterMu.Unlock()
					workerLog.WithError(err).WithField("mapId", stats.MapID).Warn("map render failed; skipping")
				default:
					rendered++
					counterMu.Unlock()
					workerLog.WithFields(logrus.Fields{
						"mapId":       stats.MapID,
						"width":       stats.Width,
						"height":      stats.Height,
						"durationMs":  stats.DurationMs,
						"spriteCount": stats.SpriteCount,
						"output":      "render.png",
					}).Info("map rendered")
				}
			}
		}(i)
	}

	for _, img := range maps {
		if ctx.Err() != nil {
			break
		}
		jobs <- img
	}
	close(jobs)
	wg.Wait()

	l.WithFields(logrus.Fields{
		"rendered": rendered,
		"skipped":  skipped,
		"failed":   failed,
	}).Info("map render batch complete")
	return nil
}

// maxPixelsFromEnv returns the cap from WZ_EXTRACT_MAX_MAP_PIXELS, or 0 to fall
// back to mapimage.DefaultMaxPixels. Non-integer values fall back to default.
func maxPixelsFromEnv(l logrus.FieldLogger) int {
	v := os.Getenv("WZ_EXTRACT_MAX_MAP_PIXELS")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		l.WithField("value", v).Warn("invalid WZ_EXTRACT_MAX_MAP_PIXELS; using default")
		return 0
	}
	return n
}
