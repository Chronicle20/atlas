package data

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

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
		defer workers.CloseMonolith()
		maxParallel := envInt("INGEST_MAX_PARALLEL", 4)

		// versionWarnOnce implements the C-5 declared-version cross-check:
		// warn (never fail) once per job when the archives' detected game
		// version disagrees with the ingest params.
		var versionWarnOnce sync.Once

		// runOne resolves the worker's archive (per-archive object or
		// monolithic Data.wz sub-view) and runs it under the given
		// (tenanted, cancellable) context. A category genuinely absent
		// from a monolithic data set (v12 has no Quest) skips that worker
		// instead of failing the whole ingest run (task-172 C-3.4).
		runOne := func(tctx context.Context, w workers.Worker) error {
			wzFile, cleanup, err := workers.OpenArchive(tctx, l, mc, p, w.ArchiveName())
			if err != nil {
				if errors.Is(err, workers.ErrCategoryAbsent) {
					l.Warnf("%s: %s absent from monolithic Data.wz — skipping worker (category not present in this data set)", w.Name(), w.ArchiveName())
					return nil
				}
				return fmt.Errorf("%s open %s: %w", w.Name(), w.ArchiveName(), err)
			}
			defer cleanup()
			if gv := wzFile.GameVersion(); gv != 0 && gv != int(p.MajorVersion) {
				versionWarnOnce.Do(func() {
					l.Warnf("WZ data declares game version %d but ingest params are %s %d.%d — check the upload landed under the intended tenant/version", gv, p.Region, p.MajorVersion, p.MinorVersion)
				})
			}
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
