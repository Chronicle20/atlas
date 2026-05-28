package storage

import (
	"context"
	"fmt"
)

// ResolveScope returns "tenants/<id>" or "shared" based on a HEAD probe of
// (tenant, region, version, subPath). `subPath` is the per-domain bucket
// suffix the caller wants to look up: character handlers pass
// "atlases/<partClass>", the map handler passes "map/<mapID>", etc.
//
// An earlier version hardcoded "atlases/" into the probe prefix, which
// silently routed every non-character ResolveScope call to "shared" — see
// map render 404s on PR-544 against tenant data that did exist.
//
// Result is cached per (tenant, region, version, subPath).
func (s *Storage) ResolveScope(ctx context.Context, tenantID, region, version, subPath string) (string, error) {
	cacheKey := tenantID + "|" + region + "|" + version + "|" + subPath
	if v, ok := s.Caches.Scope.Get(cacheKey); ok {
		return v, nil
	}
	prefix := fmt.Sprintf("tenants/%s/regions/%s/versions/%s/%s/", tenantID, region, version, subPath)
	has, err := s.MC.HasAny(ctx, s.Cfg.BucketAssets, prefix)
	if err != nil {
		return "", err
	}
	scope, shouldCache := resolveCacheGate(tenantID, has)
	if shouldCache {
		s.Caches.Scope.Add(cacheKey, scope)
	}
	// Asymmetric with positive caching: a negative ("shared") verdict is NOT
	// cached. A negative probe becomes wrong the moment ingest publishes
	// tenant data, and the cost of re-probing is one MinIO HEAD-list per
	// non-cached path (acceptable; the steady state for most maps is shared).
	// Once a tenant scope is observed it cannot become "shared" without an
	// explicit teardown flush. See task-076 F3.
	return scope, nil
}

// resolveCacheGate decides the (scope, shouldCache) tuple from a probe
// result. Positive results yield a tenant-scoped path that should be
// cached; negative results yield "shared" which is NOT cached (see F3).
func resolveCacheGate(tenantID string, has bool) (scope string, shouldCache bool) {
	if has {
		return "tenants/" + tenantID, true
	}
	return "shared", false
}
