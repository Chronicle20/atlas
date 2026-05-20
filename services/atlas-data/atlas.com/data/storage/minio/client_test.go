package minio

import "testing"

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("MINIO_BUCKET_WZ", "")
	t.Setenv("MINIO_BUCKET_ASSETS", "")
	t.Setenv("MINIO_BUCKET_RENDERS", "")
	t.Setenv("MINIO_BUCKET_CANONICAL", "")
	cfg := FromEnv()
	if cfg.BucketWZ != "atlas-wz" {
		t.Fatalf("default BucketWZ = %s", cfg.BucketWZ)
	}
	if cfg.BucketAssets != "atlas-assets" {
		t.Fatalf("default BucketAssets = %s", cfg.BucketAssets)
	}
	if cfg.BucketRenders != "atlas-renders" {
		t.Fatalf("default BucketRenders = %s", cfg.BucketRenders)
	}
	if cfg.BucketCanonical != "atlas-canonical" {
		t.Fatalf("default BucketCanonical = %s", cfg.BucketCanonical)
	}
}
