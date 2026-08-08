package ingest

import (
	"atlas-data/data"
	"atlas-data/data/workers"
	"atlas-data/ingestrun"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/sirupsen/logrus"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// Run is invoked when MODE=ingest (k8s Job pod). It reads SCOPE/REGION/version
// env vars set by JobCreator, builds workers.Params, and invokes
// data.RunWorkers. No HTTP server is started.
func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=ingest starting")
	p, err := paramsFromEnv()
	if err != nil {
		return err
	}
	// Ingest pods don't need migrations (the REST pod ran them on startup).
	db := database.Connect(l)
	mc, err := minio.NewClient(minio.FromEnv())
	if err != nil {
		return fmt.Errorf("minio init: %w", err)
	}

	// Refresh the Watchdog heartbeat every 30s while workers run, and publish
	// per-worker progress to the run record the REST pod initialised at Job
	// creation. Both are gated on the same env-derived suffix: absent SCOPE/
	// REGION means the compose / unit-test path, where neither signal has a
	// reader (PRD FR-2.6).
	var sink *redisSink
	if suffix := ingestJobSuffixFromEnv(); suffix != "" {
		rdb := redis.Connect(l)
		reg := ingestrun.NewJobRegistry(rdb)
		routine.Go(l, ctx, func(_ context.Context) { runHeartbeat(ctx, l, reg, suffix) })

		runId := ingestRunIdFromEnv()
		sink = newRedisSink(l, ingestrun.NewRunRegistry(rdb), suffix, runId)
		now := time.Now().UTC()
		sink.Init(ctx, ingestrun.NewRecord(
			runId, "", p.ScopeKey, p.Region,
			fmt.Sprintf("%d.%d", p.MajorVersion, p.MinorVersion),
			os.Getenv("TENANT_ID"), now, workers.RegisteredNames(),
		), workers.RegisteredNames(), now)
	} else {
		l.Info("ingest heartbeat and progress skipped: SCOPE/REGION/MAJOR_VERSION/MINOR_VERSION env not set (compose / test path)")
	}

	// Build the option list conditionally: passing a typed-nil *redisSink
	// through data.WithProgress would produce a non-nil interface whose
	// methods then run on a nil receiver.
	var opts []data.RunOption
	if sink != nil {
		opts = append(opts, data.WithProgress(sink))
	}
	runErr := data.RunWorkers(l, db, mc, opts...)(ctx, p)
	if sink != nil {
		sink.Finish(ctx, runErr, time.Now().UTC())
	}
	return runErr
}

func paramsFromEnv() (workers.Params, error) {
	major, err := strconv.ParseUint(os.Getenv("MAJOR_VERSION"), 10, 16)
	if err != nil {
		return workers.Params{}, fmt.Errorf("MAJOR_VERSION: %w", err)
	}
	minor, err := strconv.ParseUint(os.Getenv("MINOR_VERSION"), 10, 16)
	if err != nil {
		return workers.Params{}, fmt.Errorf("MINOR_VERSION: %w", err)
	}
	scratch := os.Getenv("SCRATCH_DIR")
	if scratch == "" {
		scratch = "/scratch"
	}
	return workers.Params{
		ScopeKey:     os.Getenv("SCOPE"),
		Region:       os.Getenv("REGION"),
		MajorVersion: uint16(major),
		MinorVersion: uint16(minor),
		ScratchDir:   scratch,
	}, nil
}
