package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
)

// GetMap fetches the layout JSON + each layer PNG for a map. Cached via
// Caches.Map keyed on (scope, region, version, mapID).
//
// scope is the resolver output (e.g. "shared" or "tenants/<id>"). The keys
// produced here mirror the layout written by atlas-data ingest:
//
//	<scope>/regions/<region>/versions/<version>/map/<mapID>/layout.json
//	<scope>/regions/<region>/versions/<version>/map/<mapID>/layers/<layer.Source>.png
//
// Layer PNGs are named by maplayout.Layer.Source — that field exists exactly
// to decouple the on-disk key from the in-memory numeric ID. atlas-data
// writes "layer-0.png", "layer-1.png" etc.; an earlier version of this
// function used "%d.png" off layer.ID and silently 404'd every layer.
func (s *Storage) GetMap(ctx context.Context, scope, region, version string, mapID uint32) (*MapEntry, error) {
	cacheKey := fmt.Sprintf("%s|%s|%s|%d", scope, region, version, mapID)
	if entry, ok := s.Caches.Map.Get(cacheKey); ok {
		return &entry, nil
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

	entry := MapEntry{Layout: layout, Layers: make(map[int][]byte, len(layout.Layers))}
	for _, layer := range layout.Layers {
		source := layer.Source
		if source == "" {
			source = fmt.Sprintf("layer-%d", layer.ID)
		}
		layerKey := fmt.Sprintf("%s/regions/%s/versions/%s/map/%d/layers/%s.png", scope, region, version, mapID, source)
		layerRC, err := s.MC.Get(ctx, s.Cfg.BucketAssets, layerKey)
		if err != nil {
			return nil, fmt.Errorf("get layer %s: %w", layerKey, err)
		}
		b, err := io.ReadAll(layerRC)
		_ = layerRC.Close()
		if err != nil {
			return nil, fmt.Errorf("read layer %s: %w", layerKey, err)
		}
		entry.Layers[layer.ID] = b
	}

	s.Caches.Map.Add(cacheKey, entry)
	return &entry, nil
}
