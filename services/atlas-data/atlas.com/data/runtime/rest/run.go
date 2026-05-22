package rest

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Run is the MODE=rest entry point. The actual /api/data/process wiring is
// installed unconditionally by main.go via InitResource so the same HTTP
// stack serves MODE=rest and MODE=all without divergence. When MODE=rest is
// set explicitly main.go takes a slimmer path: no in-process worker setup,
// no Kafka consumers. This function simply blocks until ctx is cancelled.
//
// The JobCreator/Watchdog lifecycle is wired in main.go where the JobCreator
// is constructed once and reused for both InitResource and Watchdog.Run.
func Run(ctx context.Context, l logrus.FieldLogger) error {
	l.Info("atlas-data MODE=rest starting; HTTP only, no in-process workers")
	<-ctx.Done()
	return nil
}
