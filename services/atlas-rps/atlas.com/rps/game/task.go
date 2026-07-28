package game

import (
	"atlas-rps/kafka/message/rps"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// SweepTaskName identifies the sweep task's trace span.
const SweepTaskName = "rps_sweep_task"

// SweepTask periodically reclaims RPS sessions abandoned past their TTL. It
// implements the tasks.Task interface structurally (Run + SleepTime) without
// importing the "atlas-rps/tasks" package, mirroring
// atlas-expressions/atlas.com/expressions/expression/task.go's RevertTask.
//
// A swept session is disposed with NO payout: PopExpired has already removed
// it from the registry, so Run emits the same GameEnded{disconnected} event
// Processor.Dispose would buffer, directly via the producer (there is no
// session left in the registry for a Processor.Dispose call to find, and no
// payout saga is ever built or submitted for a sweep).
type SweepTask struct {
	l        logrus.FieldLogger
	interval time.Duration
}

// NewSweepTask creates a new SweepTask that runs every interval.
func NewSweepTask(l logrus.FieldLogger, interval time.Duration) *SweepTask {
	l.Infof("Initializing RPS session sweep task to run every %dms", interval.Milliseconds())
	return &SweepTask{l, interval}
}

// Run pops every expired session across all tracked tenants and disposes
// each with no payout, re-injecting the swept model's tenant onto the
// context so the emitted event carries the correct tenant headers.
func (s *SweepTask) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-rps").Start(context.Background(), SweepTaskName)
	defer span.End()

	for _, m := range GetRegistry().PopExpired(sctx) {
		tctx := tenant.WithContext(sctx, m.Tenant())
		if err := producer.ProviderImpl(s.l)(tctx)(rps.EnvEventTopic)(gameEndedEventProvider(m.CharacterId(), m.WorldId(), m.ChannelId(), rps.ReasonDisconnected, nil)); err != nil {
			s.l.WithError(err).Errorf("Unable to emit GameEnded for swept RPS session for character [%d].", m.CharacterId())
		}
	}
}

// SleepTime returns the configured interval between sweep runs.
func (s *SweepTask) SleepTime() time.Duration {
	return s.interval
}
