package data

import (
	"context"
	"fmt"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"

	minio "atlas-data/storage/minio"
)

// FetchAndOpen downloads bucket/key into scratchDir and returns the parsed WZ
// File along with the on-disk path for cleanup. The caller MUST call
// (*wz.File).Close() and remove the localPath when done.
//
// WZ region/encryption and version are auto-detected by wz.Open.
func FetchAndOpen(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, bucket, key, scratchDir string) (*wz.File, string, error) {
	localPath, err := mc.DownloadToScratch(ctx, bucket, key, scratchDir)
	if err != nil {
		return nil, "", fmt.Errorf("download %s/%s: %w", bucket, key, err)
	}
	file, err := wz.Open(l, localPath)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, "", fmt.Errorf("open wz %s: %w", localPath, err)
	}
	return file, localPath, nil
}
