package storage

import (
	"context"
	"errors"
	"io"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MC is a thin wrapper around the MinIO client exposing the small surface
// atlas-renders needs (Get / Put / Stat / HasAny).
type MC struct {
	mc *miniogo.Client
}

func NewMC(cfg Config) (*MC, error) {
	mc, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MC{mc: mc}, nil
}

// Get returns a reader for the object at (bucket, key). The minio-go client
// lazily defers the actual request: mc.GetObject succeeds with a non-nil
// *Object even when the key doesn't exist, and the NoSuchKey error only
// surfaces on the FIRST Read. Callers that copy directly into an
// http.ResponseWriter after setting the Content-Type header would otherwise
// send a 200 + 0-byte response on a cache miss (observed against the
// renders bucket on PR-544). Stat first so a missing object surfaces as an
// error here instead of later as silent corruption.
func (m *MC) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := m.mc.GetObject(ctx, bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, statErr := obj.Stat(); statErr != nil {
		_ = obj.Close()
		return nil, statErr
	}
	return obj, nil
}

func (m *MC) Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	_, err := m.mc.PutObject(ctx, bucket, key, r, size, miniogo.PutObjectOptions{ContentType: contentType})
	return err
}

func (m *MC) Stat(ctx context.Context, bucket, key string) (bool, error) {
	_, err := m.mc.StatObject(ctx, bucket, key, miniogo.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	var er miniogo.ErrorResponse
	if errors.As(err, &er) && er.Code == "NoSuchKey" {
		return false, nil
	}
	return false, err
}

// FGet downloads (bucket, key) to localPath. Used by the WZ archive cache
// where the file must live on a local filesystem so wz.Open can read it via
// positional ReadAt for the lifetime of the *wz.File. Returns an error if
// the object doesn't exist (no silent zero-byte file).
func (m *MC) FGet(ctx context.Context, bucket, key, localPath string) error {
	return m.mc.FGetObject(ctx, bucket, key, localPath, miniogo.GetObjectOptions{})
}

// HasAny returns true if any object exists under the supplied prefix. The
// scope resolver uses this as a one-item ListObjects probe.
func (m *MC) HasAny(ctx context.Context, bucket, prefix string) (bool, error) {
	ch := m.mc.ListObjects(ctx, bucket, miniogo.ListObjectsOptions{Prefix: prefix, MaxKeys: 1, Recursive: true})
	for obj := range ch {
		if obj.Err != nil {
			return false, obj.Err
		}
		return true, nil
	}
	return false, nil
}
