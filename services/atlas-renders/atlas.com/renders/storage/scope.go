package storage

import (
	"context"
	"fmt"
)

// ResolveScope returns "tenants/<id>" or "shared" based on a HEAD probe.
// Result is cached per (tenant, region, version, partClass).
func (s *Storage) ResolveScope(ctx context.Context, tenantID, region, version, partClass string) (string, error) {
	cacheKey := tenantID + "|" + region + "|" + version + "|" + partClass
	if v, ok := s.Caches.Scope.Get(cacheKey); ok {
		return v, nil
	}
	// Probe the first id under tenant/<>/atlases/<partClass>/. Use ListObjects with limit 1.
	prefix := fmt.Sprintf("tenants/%s/regions/%s/versions/%s/atlases/%s/", tenantID, region, version, partClass)
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
