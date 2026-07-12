package tasks

import (
	"context"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/sirupsen/logrus"
)

type Task interface {
	Run()

	SleepTime() time.Duration
}

func Register(l logrus.FieldLogger, ctx context.Context) func(t Task) {
	return func(t Task) {
		routine.Go(l, ctx, func(_ context.Context) {
			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				case <-time.After(t.SleepTime()):
					t.Run()
				}
			}
		})
	}
}
