package data

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"atlas-data/data/workers"
	minio "atlas-data/storage/minio"
)

// RunWorkers fans out the registered Worker set over the WZ archives stored in
// MinIO at "<scope>/regions/<region>/versions/<major>.<minor>/<archive>".
// Each worker is invoked with a context that already carries the per-tenant
// model so worker bodies don't have to redo (and possibly mis-handle) the
// injection. See workers.WithTenant — pre-injecting here means the discard
// pattern that bit Commodity (df89b8bee) is impossible to trigger downstream.
func RunWorkers(l logrus.FieldLogger, db *gorm.DB, mc *minio.Client) func(ctx context.Context, p workers.Params) error {
	return func(ctx context.Context, p workers.Params) error {
		maxParallel := envInt("INGEST_MAX_PARALLEL", 4)
		sem := semaphore.NewWeighted(int64(maxParallel))
		g, gctx := errgroup.WithContext(ctx)
		tenantedCtx, _, terr := workers.WithTenant(gctx, p)
		if terr != nil {
			return fmt.Errorf("ingest withTenant: %w", terr)
		}
		for _, w := range workers.Registered {
			w := w
			g.Go(func() error {
				if err := sem.Acquire(gctx, 1); err != nil {
					return err
				}
				defer sem.Release(1)
				wzKey := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion, w.ArchiveName())
				wzFile, localPath, err := FetchAndOpen(gctx, l, mc, mc.Cfg().BucketWZ, wzKey, p.ScratchDir)
				if err != nil {
					return fmt.Errorf("%s open %s: %w", w.Name(), wzKey, err)
				}
				defer func() {
					wzFile.Close()
					_ = os.Remove(localPath)
				}()
				return w.Run(tenantedCtx, l, db, mc, wzFile, p)
			})
		}
		return g.Wait()
	}
}

func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			return x
		}
	}
	return d
}
