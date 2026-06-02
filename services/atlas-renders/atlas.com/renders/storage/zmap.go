package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// GetZmap fetches the character-meta/zmap.json sidecar emitted by the
// atlas-data Character worker. The zmap is an ordered list of layer names
// (Base.wz/zmap.img declaration order = render order, front-to-back). The
// character compositor resolves each sprite's render-layer label
// (manifest.Sprite.Z, the canvas `z` child — NOT Part, the canvas name) to its
// index in this list and draws back-most first.
//
// The key shape in MinIO mirrors smap.json (same character-meta dir):
//
//	<scope>/regions/<region>/versions/<version>/character-meta/zmap.json
//
// Cached via Caches.Zmap so re-renders avoid the round trip; the cache key is
// "<scope>|<region>|<version>". The scope is resolved by ResolveSmapScope —
// zmap.json and smap.json are co-located and emitted atomically in the same
// ingest step, so the smap-scope probe is authoritative for both.
func (s *Storage) GetZmap(ctx context.Context, scope, region, version string) ([]string, error) {
	cacheKey := scope + "|" + region + "|" + version
	if entry, ok := s.Caches.Zmap.Get(cacheKey); ok {
		return entry, nil
	}

	key := fmt.Sprintf("%s/regions/%s/versions/%s/character-meta/zmap.json", scope, region, version)
	rc, err := s.MC.Get(ctx, s.Cfg.BucketAssets, key)
	if err != nil {
		return nil, fmt.Errorf("get zmap %s: %w", key, err)
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read zmap %s: %w", key, err)
	}

	var out []string
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode zmap %s: %w", key, err)
	}
	s.Caches.Zmap.Add(cacheKey, out)
	return out, nil
}
