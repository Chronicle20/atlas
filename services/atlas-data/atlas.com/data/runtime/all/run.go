package all

import (
	"context"

	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=all starting; HTTP + in-process workers")
	// Stub — wraps the existing main flow. Filled in Task 8.
	return nil
}
