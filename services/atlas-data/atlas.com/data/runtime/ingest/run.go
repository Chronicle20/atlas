package ingest

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"atlas-data/data"
	"atlas-data/data/workers"
	minio "atlas-data/storage/minio"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/sirupsen/logrus"
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
	return data.RunWorkers(l, db, mc)(ctx, p)
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
