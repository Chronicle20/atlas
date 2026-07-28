package minioreconcile

import (
	"context"
	"strings"
	"time"

	minio "atlas-data/storage/minio"
)

// minioStore adapts *minio.Client to Store.
type minioStore struct{ mc *minio.Client }

// NewStore returns a Store backed by the real MinIO client.
func NewStore(mc *minio.Client) Store { return minioStore{mc: mc} }

func (s minioStore) Buckets() []string {
	c := s.mc.Cfg()
	return []string{c.BucketWZ, c.BucketAssets, c.BucketRenders}
}

func (s minioStore) ListTenantIDs(ctx context.Context, bucket string) ([]string, error) {
	keys, err := s.mc.ListTenantPrefixes(ctx, bucket)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		if id := parseTenantID(k); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s minioStore) PrefixInfo(ctx context.Context, bucket, prefix string) (PrefixInfo, error) {
	objs, err := s.mc.List(ctx, bucket, prefix)
	if err != nil {
		return PrefixInfo{}, err
	}
	var info PrefixInfo
	var newest time.Time
	for _, o := range objs {
		info.Count++
		info.Bytes += o.Size
		if o.LastModified.After(newest) {
			newest = o.LastModified
		}
	}
	info.Newest = newest
	return info, nil
}

func (s minioStore) RemovePrefix(ctx context.Context, bucket, prefix string) error {
	return s.mc.RemovePrefix(ctx, bucket, prefix)
}

// parseTenantID extracts the uuid from a "tenants/<uuid>/" key. Returns "" when
// the key is not a tenant child prefix.
func parseTenantID(key string) string {
	rest := strings.TrimPrefix(key, "tenants/")
	if rest == key { // no "tenants/" prefix
		return ""
	}
	i := strings.IndexByte(rest, '/')
	if i <= 0 {
		return ""
	}
	return rest[:i]
}
