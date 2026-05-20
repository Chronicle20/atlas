package minio

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// DownloadToScratch fetches bucket/key into scratchDir and returns the local path.
func (c *Client) DownloadToScratch(ctx context.Context, bucket, key, scratchDir string) (string, error) {
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		return "", err
	}
	local := filepath.Join(scratchDir, filepath.Base(key))
	rc, err := c.Get(ctx, bucket, key)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	f, err := os.Create(local)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, rc); err != nil {
		return "", err
	}
	return local, nil
}
