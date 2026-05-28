package mapimage

import (
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"strings"
)

// Index groups the Back/Tile/Obj lookup maps plus per-mapId map images
// for fast resolution during compositing. All references live in the same
// Map.wz file handle (Index.File).
type Index struct {
	File *wz.File
	back map[string]*wz.Image
	tile map[string]*wz.Image
	obj  map[string]*wz.Image
	maps map[string]*wz.Image
}

// NewIndex constructs the lookup maps for Back, Tile, Obj, and Map sub-directories.
// Passing a file that is not `Map.wz` returns an Index with empty maps.
func NewIndex(f *wz.File) *Index {
	idx := &Index{
		File: f,
		back: map[string]*wz.Image{},
		tile: map[string]*wz.Image{},
		obj:  map[string]*wz.Image{},
		maps: map[string]*wz.Image{},
	}
	if f == nil {
		return idx
	}
	root := f.Root()
	if root == nil {
		return idx
	}
	for _, d := range root.Directories() {
		switch d.Name() {
		case "Back":
			buildImageSet(d, idx.back)
		case "Tile":
			buildImageSet(d, idx.tile)
		case "Obj":
			buildImageSet(d, idx.obj)
		case "Map":
			for _, sub := range d.Directories() {
				for _, img := range sub.Images() {
					idx.maps[img.Name()] = img
				}
			}
		}
	}
	return idx
}

// Maps returns the per-map image lookup, keyed by map id (e.g. "100000000").
func (i *Index) Maps() map[string]*wz.Image {
	return i.maps
}

func buildImageSet(d *wz.Directory, dst map[string]*wz.Image) {
	for _, img := range d.Images() {
		dst[strings.ToLower(img.Name())] = img
	}
}
