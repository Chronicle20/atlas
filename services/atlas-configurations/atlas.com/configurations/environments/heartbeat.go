package environments

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	envlib "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// StartHeartbeat republishes the baseline environment record every 30s.
// Consumers use the arrival of ANY message on the topic as liveness, so the
// payload is deliberately the unchanged baseline record: compaction keeps
// exactly one copy per key regardless of how often it is written (design
// §4.3, Task 18's StaleAfter).
func StartHeartbeat(l logrus.FieldLogger, ctx context.Context, p Processor) {
	routine.Go(l, ctx, func(_ context.Context) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.Republish(envlib.Self()); err != nil {
					l.WithError(err).Warn("environment heartbeat failed")
				}
			}
		}
	})
}
