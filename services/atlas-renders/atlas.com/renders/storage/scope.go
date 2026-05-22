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
	scope := "shared"
	if has {
		scope = "tenants/" + tenantID
	}
	s.Caches.Scope.Add(cacheKey, scope)
	return scope, nil
}
