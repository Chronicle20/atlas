package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// GetSmap fetches the character-meta/smap.json sidecar emitted by the
// atlas-data Character worker. The smap is a flat layer-name → slot-codes
// string map (e.g. "cap" → "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe") that
// atlas-renders consumes to drive equipment-vs-hair occlusion in the
// character composite (donor: characterimage/vslot.go).
//
// The key shape in MinIO mirrors the donor on-disk layout the wz-extractor
// used to write:
//
//	<scope>/regions/<region>/versions/<version>/character-meta/smap.json
//
// Cached via Caches.Smap so re-renders avoid the round trip; the cache key
// is "<scope>|<region>|<version>". A nil-valued cache hit means a previous
// fetch concluded the sidecar is absent (we cache the negative result so
// renders aren't held back by repeated 404 probes — see GetSmap docstring
// in the caller wiring for fallback behaviour).
func (s *Storage) GetSmap(ctx context.Context, scope, region, version string) (map[string]string, error) {
	cacheKey := scope + "|" + region + "|" + version
	if entry, ok := s.Caches.Smap.Get(cacheKey); ok {
		return entry, nil
	}

	key := fmt.Sprintf("%s/regions/%s/versions/%s/character-meta/smap.json", scope, region, version)
	rc, err := s.MC.Get(ctx, s.Cfg.BucketAssets, key)
	if err != nil {
		return nil, fmt.Errorf("get smap %s: %w", key, err)
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read smap %s: %w", key, err)
	}

	out := map[string]string{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode smap %s: %w", key, err)
	}
	s.Caches.Smap.Add(cacheKey, out)
	return out, nil
}

// ResolveSmapScope returns the MinIO scope ("tenants/<id>" or "shared")
// under which the character-meta/smap.json sidecar lives. Mirrors
// ResolveScope's semantics but probes the smap.json key directly rather
// than an atlases/<partClass>/ prefix — there's no partClass split for the
// smap sidecar.
//
// The resolved scope is cached under the synthetic key
// "<tenant>|<region>|<version>|character-meta" inside Caches.Scope so the
// fast path for repeated renders matches the per-partClass atlas scope
// lookups.
func (s *Storage) ResolveSmapScope(ctx context.Context, tenantID, region, version string) (string, error) {
	cacheKey := tenantID + "|" + region + "|" + version + "|character-meta"
	if v, ok := s.Caches.Scope.Get(cacheKey); ok {
		return v, nil
	}
	tenantKey := fmt.Sprintf("tenants/%s/regions/%s/versions/%s/character-meta/smap.json", tenantID, region, version)
	if has, err := s.MC.Stat(ctx, s.Cfg.BucketAssets, tenantKey); err == nil && has {
		s.Caches.Scope.Add(cacheKey, "tenants/"+tenantID)
		return "tenants/" + tenantID, nil
	}
	// Fall back to shared scope. We don't HEAD-probe shared first because
	// almost every deployment relies on the shared sidecar — a miss here is
	// recoverable (atlas-renders disables occlusion and logs a warning).
	//
	// We also don't cache this verdict — see F3 in
	// docs/tasks/task-076-task071-followups: a negative probe becomes wrong
	// the moment ingest publishes the smap sidecar for this tenant.
	return "shared", nil
}
