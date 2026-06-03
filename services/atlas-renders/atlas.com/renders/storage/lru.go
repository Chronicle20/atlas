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

// MapEntry is a hot-cached map layout. atlas-renders now composites layers
// directly from Map.wz at render time (see Storage.WZ / wzcache.go), so this
// entry no longer carries pre-rendered layer bytes — the layout alone
// suffices for cache hits.
type MapEntry struct {
	Layout maplayout.Layout
}

// Caches holds the LRU caches atlas-renders uses to short-circuit MinIO
// round-trips on the render path.
type Caches struct {
	Atlas *lru.Cache[string, AtlasEntry]
	Map   *lru.Cache[string, MapEntry]
	Scope *lru.Cache[string, string]
	// Smap caches the per-(scope, region, version) character-meta/smap.json
	// payload (layer-name → slot-codes string). The atlas-data Character
	// worker emits one smap.json per ingest; on read it's a near-singleton —
	// a 16-entry cache covers multi-tenant deployments comfortably.
	Smap *lru.Cache[string, map[string]string]
	// Zmap caches the per-(scope, region, version) character-meta/zmap.json
	// payload (ordered layer-name list = render order). Same cardinality and
	// lifecycle as Smap — one payload per active tenant version.
	Zmap *lru.Cache[string, []string]
}

// NewCaches allocates the LRU caches. Sizes are tunables; the design suggests
// 256/64/1024 for atlas/map/scope and a small 16 each for smap and zmap (one
// payload per active tenant version).
func NewCaches(atlasSize, mapSize, scopeSize int) *Caches {
	a, _ := lru.New[string, AtlasEntry](atlasSize)
	m, _ := lru.New[string, MapEntry](mapSize)
	s, _ := lru.New[string, string](scopeSize)
	sm, _ := lru.New[string, map[string]string](16)
	zm, _ := lru.New[string, []string](16)
	return &Caches{Atlas: a, Map: m, Scope: s, Smap: sm, Zmap: zm}
}
