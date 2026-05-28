package atlas

import (
	"fmt"
	"image"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
)

// Input is a single sprite to pack.
type Input struct {
	Name    string
	Img     image.Image
	Origin  image.Point
	Anchors map[string]image.Point
	Z       int
}

// Pack lays sprites out using MaxRects with Best-Short-Side-Fit, grows the bin
// in powers of two from 256 to 4096, and emits a deterministic sheet+manifest.
func Pack(in []Input) (image.Image, manifest.Manifest, error) {
	if len(in) == 0 {
		return nil, manifest.Manifest{}, fmt.Errorf("atlas.Pack: empty input")
	}
	// Stable pre-sort: (width desc, height desc, name asc).
	sorted := make([]Input, len(in))
	copy(sorted, in)
	sort.SliceStable(sorted, func(i, j int) bool {
		wi, hi := sorted[i].Img.Bounds().Dx(), sorted[i].Img.Bounds().Dy()
		wj, hj := sorted[j].Img.Bounds().Dx(), sorted[j].Img.Bounds().Dy()
		if wi != wj {
			return wi > wj
		}
		if hi != hj {
			return hi > hj
		}
		return sorted[i].Name < sorted[j].Name
	})

	for size := 256; size <= 4096; size *= 2 {
		sheet, m, ok := tryPack(sorted, size)
		if ok {
			return sheet, m, nil
		}
	}
	return nil, manifest.Manifest{}, fmt.Errorf("atlas.Pack: sprites do not fit in 4096x4096")
}
