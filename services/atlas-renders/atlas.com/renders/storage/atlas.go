package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
)

// GetAtlas fetches the (PNG + Manifest) pair for one part atlas keyed by
// (scope, region, version, partClass, id) and caches the result in
// Caches.Atlas so subsequent renders avoid the round trip. Scope is the
// "tenants/<id>" or "shared" prefix returned by ResolveScope.
//
// The MinIO layout (per design §4.2) is:
//
//	<scope>/regions/<region>/versions/<version>/atlases/<partClass>/<id>.png
//	<scope>/regions/<region>/versions/<version>/atlases/<partClass>/<id>.json
//
// PNG decoding is left to the caller; this method only streams the bytes.
func (s *Storage) GetAtlas(ctx context.Context, scope, region, version, partClass string, id uint32) (*AtlasEntry, error) {
	cacheKey := fmt.Sprintf("%s|%s|%s|%s|%d", scope, region, version, partClass, id)
	if entry, ok := s.Caches.Atlas.Get(cacheKey); ok {
		return &entry, nil
	}

	pngKey := fmt.Sprintf("%s/regions/%s/versions/%s/atlases/%s/%d.png", scope, region, version, partClass, id)
	manKey := fmt.Sprintf("%s/regions/%s/versions/%s/atlases/%s/%d.json", scope, region, version, partClass, id)

	pngRC, err := s.MC.Get(ctx, s.Cfg.BucketAssets, pngKey)
	if err != nil {
		return nil, fmt.Errorf("get atlas png %s: %w", pngKey, err)
	}
	defer pngRC.Close()
	pngBytes, err := io.ReadAll(pngRC)
	if err != nil {
		return nil, fmt.Errorf("read atlas png %s: %w", pngKey, err)
	}

	manRC, err := s.MC.Get(ctx, s.Cfg.BucketAssets, manKey)
	if err != nil {
		return nil, fmt.Errorf("get atlas manifest %s: %w", manKey, err)
	}
	defer manRC.Close()
	manBytes, err := io.ReadAll(manRC)
	if err != nil {
		return nil, fmt.Errorf("read atlas manifest %s: %w", manKey, err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manBytes, &m); err != nil {
		return nil, fmt.Errorf("decode manifest %s: %w", manKey, err)
	}

	entry := AtlasEntry{PNG: pngBytes, Manifest: m}
	s.Caches.Atlas.Add(cacheKey, entry)
	return &entry, nil
}
