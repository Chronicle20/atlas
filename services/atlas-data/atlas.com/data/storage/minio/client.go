package minio

import (
	"context"
	"errors"
	"io"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	cfg Config
	mc  *miniogo.Client
}

func NewClient(cfg Config) (*Client, error) {
	mc, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Client{cfg: cfg, mc: mc}, nil
}

func (c *Client) Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, bucket, key, r, size, miniogo.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, bucket, key, miniogo.GetObjectOptions{})
	return obj, err
}

func (c *Client) Stat(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.mc.StatObject(ctx, bucket, key, miniogo.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	var er miniogo.ErrorResponse
	if errors.As(err, &er) && er.Code == "NoSuchKey" {
		return false, nil
	}
	return false, err
}

func (c *Client) RemovePrefix(ctx context.Context, bucket, prefix string) error {
	objCh := c.mc.ListObjects(ctx, bucket, miniogo.ListObjectsOptions{Prefix: prefix, Recursive: true})
	errCh := c.mc.RemoveObjects(ctx, bucket, objCh, miniogo.RemoveObjectsOptions{})
	for e := range errCh {
		if e.Err != nil {
			return e.Err
		}
	}
	return nil
}

func (c *Client) Cfg() Config { return c.cfg }

// Stats reports aggregate information about objects under a prefix.
type Stats struct {
	Count     int
	Size      int64
	UpdatedAt string // RFC3339 of the latest LastModified, or empty if no objects
}

// PrefixStats walks every object under bucket/prefix and returns the count,
// total size, and most recent LastModified timestamp.
func (c *Client) PrefixStats(ctx context.Context, bucket, prefix string) (Stats, error) {
	var s Stats
	ch := c.mc.ListObjects(ctx, bucket, miniogo.ListObjectsOptions{Prefix: prefix, Recursive: true})
	var latest time.Time
	for obj := range ch {
		if obj.Err != nil {
			return Stats{}, obj.Err
		}
		s.Count++
		s.Size += obj.Size
		if obj.LastModified.After(latest) {
			latest = obj.LastModified
		}
	}
	if !latest.IsZero() {
		s.UpdatedAt = latest.UTC().Format(time.RFC3339)
	}
	return s, nil
}
