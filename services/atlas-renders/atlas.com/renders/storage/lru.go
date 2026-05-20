package storage

import (
	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
	lru "github.com/hashicorp/golang-lru/v2"
)

// AtlasEntry is a hot-cached sprite atlas: the PNG bytes plus its manifest.
type AtlasEntry struct {
	PNG      []byte
	Manifest manifest.Manifest
}

// MapEntry is a hot-cached map composite source: the per-layer PNG bytes plus
// the parsed Map.img layout describing how to blit them.
type MapEntry struct {
	Layers map[int][]byte
	Layout maplayout.Layout
}

// Caches holds the LRU caches atlas-renders uses to short-circuit MinIO
// round-trips on the render path.
type Caches struct {
	Atlas *lru.Cache[string, AtlasEntry]
	Map   *lru.Cache[string, MapEntry]
	Scope *lru.Cache[string, string]
}

// NewCaches allocates the three LRU caches. Sizes are tunables; the design
// suggests 256/64/1024 for atlas/map/scope respectively.
func NewCaches(atlasSize, mapSize, scopeSize int) *Caches {
	a, _ := lru.New[string, AtlasEntry](atlasSize)
	m, _ := lru.New[string, MapEntry](mapSize)
	s, _ := lru.New[string, string](scopeSize)
	return &Caches{Atlas: a, Map: m, Scope: s}
}
