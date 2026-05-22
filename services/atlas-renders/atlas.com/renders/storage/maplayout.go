package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
)

// GetMapLayout fetches just the layout JSON for a map. The legacy GetMap
// variant also loaded per-layer PNGs; those are no longer produced by ingest
// (atlas-renders composites layers from Map.wz at render time — see
// docs/tasks/task-071-.../lazy-map-render.md). Result is cached in
// Caches.Map keyed on (scope, region, version, mapID).
//
// scope is the resolver output (e.g. "shared" or "tenants/<id>"). The key
// shape matches what atlas-data ingest writes:
//
//	<scope>/regions/<region>/versions/<version>/map/<mapID>/layout.json
func (s *Storage) GetMapLayout(ctx context.Context, scope, region, version string, mapID uint32) (*maplayout.Layout, error) {
	cacheKey := fmt.Sprintf("%s|%s|%s|%d", scope, region, version, mapID)
	if entry, ok := s.Caches.Map.Get(cacheKey); ok {
		layout := entry.Layout
		return &layout, nil
	}

	layoutKey := fmt.Sprintf("%s/regions/%s/versions/%s/map/%d/layout.json", scope, region, version, mapID)
	layoutRC, err := s.MC.Get(ctx, s.Cfg.BucketAssets, layoutKey)
	if err != nil {
		return nil, fmt.Errorf("get layout %s: %w", layoutKey, err)
	}
	defer layoutRC.Close()
	layoutBytes, err := io.ReadAll(layoutRC)
	if err != nil {
		return nil, fmt.Errorf("read layout %s: %w", layoutKey, err)
	}
	var layout maplayout.Layout
	if err := json.Unmarshal(layoutBytes, &layout); err != nil {
		return nil, fmt.Errorf("decode layout %s: %w", layoutKey, err)
	}

	s.Caches.Map.Add(cacheKey, MapEntry{Layout: layout})
	return &layout, nil
}
