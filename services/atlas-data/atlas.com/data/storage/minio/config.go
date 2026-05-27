package minio

import "os"

type Config struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	BucketWZ        string
	BucketAssets    string
	BucketRenders   string
	BucketCanonical string
	UseSSL          bool
}

func FromEnv() Config {
	return Config{
		Endpoint:        os.Getenv("MINIO_ENDPOINT"),
		AccessKey:       os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey:       os.Getenv("MINIO_SECRET_KEY"),
		BucketWZ:        envOr("MINIO_BUCKET_WZ", "atlas-wz"),
		BucketAssets:    envOr("MINIO_BUCKET_ASSETS", "atlas-assets"),
		BucketRenders:   envOr("MINIO_BUCKET_RENDERS", "atlas-renders"),
		BucketCanonical: envOr("MINIO_BUCKET_CANONICAL", "atlas-canonical"),
		UseSSL:          os.Getenv("MINIO_USE_SSL") == "true",
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
