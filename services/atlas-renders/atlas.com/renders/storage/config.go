package storage

import "os"

type Config struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	BucketAssets  string
	BucketRenders string
	BucketWZ      string
	UseSSL        bool
	// WZScratchDir is the local filesystem path where atlas-renders caches
	// downloaded *.wz files so wz.Open can read them via positional ReadAt
	// (the parser keeps the file handle open for the lifetime of the
	// *wz.File). Default "/scratch/wz" matches the emptyDir mount declared
	// in deploy/k8s/base/atlas-renders.yaml.
	WZScratchDir string
}

func ConfigFromEnv() Config {
	return Config{
		Endpoint:      os.Getenv("MINIO_ENDPOINT"),
		AccessKey:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey:     os.Getenv("MINIO_SECRET_KEY"),
		BucketAssets:  envOr("MINIO_BUCKET_ASSETS", "atlas-assets"),
		BucketRenders: envOr("MINIO_BUCKET_RENDERS", "atlas-renders"),
		BucketWZ:      envOr("MINIO_BUCKET_WZ", "atlas-wz"),
		UseSSL:        os.Getenv("MINIO_USE_SSL") == "true",
		WZScratchDir:  envOr("WZ_SCRATCH_DIR", "/scratch/wz"),
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
