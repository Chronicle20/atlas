package ingest

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Run is invoked when MODE=ingest. The actual fan-out lives in package data
// (RunWorkers) and is invoked by the future Task 12 Job-side launcher.
// For now this is a placeholder that exits when ctx is done.
func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=ingest starting; awaiting Task 12 launcher signal")
	<-ctx.Done()
	return nil
}
