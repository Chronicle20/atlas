package ingest

import (
	"context"

	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=ingest starting; workers only, no HTTP")
	// TODO Task 8: invoke workers fan-out from env.
	return nil
}
