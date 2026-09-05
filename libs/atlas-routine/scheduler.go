package routine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// drainTimeout bounds how long shutdown waits for an in-flight Run to
// return before the scheduler releases the teardown WaitGroup and lets the
// process exit anyway. A var, not a const, only so the scheduler's own
// tests can shorten it; nothing outside this package may set it.
var drainTimeout = 5 * time.Second

// minSleep is the floor applied to a non-positive SleepTime().
var minSleep = 1 * time.Second

// Task is a unit of periodic work. Run receives the scheduler loop's
// context: it is cancelled at shutdown, so a long sweep can abort mid-work
// rather than only between ticks.
type Task interface {
	Run(ctx context.Context)
	SleepTime() time.Duration
}

// Register starts t's periodic loop and returns immediately. The loop is
// sleep-first: the first Run happens one SleepTime() after registration.
// The loop returns when ctx is cancelled.
//
// wg is the teardown WaitGroup (service.Runtime.WaitGroup()). Register
// increments it before returning -- so Register MUST be called
// synchronously from main, never from inside a routine.Go -- and releases
// it when the loop returns, or drainTimeout after cancellation if an
// in-flight Run has not returned by then.
func Register(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(t Task) {
	return func(t Task) {
		wg.Add(1)
		done := make(chan struct{})

		Go(l, ctx, func(_ context.Context) {
			defer close(done)
			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				case <-time.After(sleepTimeOf(l, t)):
					t.Run(ctx)
				}
			}
		})

		Go(l, ctx, func(_ context.Context) {
			defer wg.Done()
			select {
			case <-done:
			case <-ctx.Done():
				select {
				case <-done:
				case <-time.After(drainTimeout):
					l.WithField("task", fmt.Sprintf("%T", t)).
						Warnf("Task did not return within %s of shutdown; abandoning drain.", drainTimeout)
				}
			}
		})
	}
}

func sleepTimeOf(l logrus.FieldLogger, t Task) time.Duration {
	d := t.SleepTime()
	if d > 0 {
		return d
	}
	l.WithField("task", fmt.Sprintf("%T", t)).
		Warnf("Task reported a non-positive SleepTime (%s); clamping to %s.", d, minSleep)
	return minSleep
}
