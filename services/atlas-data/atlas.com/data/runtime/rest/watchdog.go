package rest

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Watchdog periodically sweeps the set of active ingest Jobs and marks any
// that have exceeded TimeoutSecs without progress. The sweep implementation
// is a Task 12 follow-up; this struct exists so the lifecycle wiring in
// main.go is structurally complete.
type Watchdog struct {
	L           logrus.FieldLogger
	JobCreator  *JobCreator
	TimeoutSecs int
}

// Run blocks until ctx is cancelled, ticking once every 30 seconds.
func (w Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweep(ctx)
		}
	}
}

func (w Watchdog) sweep(ctx context.Context) {
	// TODO Task 12 follow-up: list Jobs by label selector, compare against
	// Redis-tracked updatedAt, mark stuck Jobs and emit a metric.
	_ = ctx
}
