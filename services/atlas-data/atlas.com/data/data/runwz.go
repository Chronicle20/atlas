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
// prerequisiteWorkerName is the worker that must finish before the parallel
// fan-out. The String worker (String.wz) populates the item-name registry that
// the Item worker resolves item/pet names from during ingest. Running them
// concurrently is an order race that leaves item/pet names empty.
const prerequisiteWorkerName = "STRING"

// splitPrerequisites partitions the registered workers into those that must run
// to completion before the rest (prerequisites) and the remaining parallel set.
func splitPrerequisites(registered []workers.Worker) (prereq, rest []workers.Worker) {
	for _, w := range registered {
		if w.Name() == prerequisiteWorkerName {
			prereq = append(prereq, w)
		} else {
			rest = append(rest, w)
		}
	}
	return prereq, rest
}

func RunWorkers(l logrus.FieldLogger, db *gorm.DB, mc *minio.Client) func(ctx context.Context, p workers.Params) error {
	return func(ctx context.Context, p workers.Params) error {
		maxParallel := envInt("INGEST_MAX_PARALLEL", 4)

		// runOne fetches the worker's archive and runs it under the given
		// (tenanted, cancellable) context.
		runOne := func(tctx context.Context, w workers.Worker) error {
			wzKey := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion, w.ArchiveName())
			wzFile, localPath, err := FetchAndOpen(tctx, l, mc, mc.Cfg().BucketWZ, wzKey, p.ScratchDir)
			if err != nil {
				return fmt.Errorf("%s open %s: %w", w.Name(), wzKey, err)
			}
			defer func() {
				wzFile.Close()
				_ = os.Remove(localPath)
			}()
			return w.Run(tctx, l, db, mc, wzFile, p)
		}

		prereq, rest := splitPrerequisites(workers.Registered)

		// Phase 1: prerequisites, sequentially, to completion. This populates the
		// item-name registry before any worker that resolves names from it runs,
		// fixing the ingest-order race that left item/pet names empty.
		pctx, _, perr := workers.WithTenant(ctx, p)
		if perr != nil {
			return fmt.Errorf("ingest withTenant: %w", perr)
		}
		for _, w := range prereq {
			if err := runOne(pctx, w); err != nil {
				return err
			}
		}

		// Phase 2: the remaining workers fan out in parallel.
		sem := semaphore.NewWeighted(int64(maxParallel))
		g, gctx := errgroup.WithContext(ctx)
		tenantedCtx, _, terr := workers.WithTenant(gctx, p)
		if terr != nil {
			return fmt.Errorf("ingest withTenant: %w", terr)
		}
		for _, w := range rest {
			w := w
			g.Go(func() error {
				if err := sem.Acquire(gctx, 1); err != nil {
					return err
				}
				defer sem.Release(1)
				return runOne(tenantedCtx, w)
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
