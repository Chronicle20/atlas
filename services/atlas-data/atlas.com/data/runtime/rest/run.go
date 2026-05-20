package rest

import (
	"context"

	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=rest starting; HTTP only, no in-process workers")
	// TODO Task 12: wire HTTP server + Job-create handlers.
	<-ctx.Done()
	return nil
}
