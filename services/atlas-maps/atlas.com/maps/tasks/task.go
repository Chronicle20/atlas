package tasks

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

type Task interface {
	Run()

	SleepTime() time.Duration
}

func Register(l logrus.FieldLogger, ctx context.Context) func(t Task) {
	return func(t Task) {
		routine.Go(l, ctx, func(_ context.Context) {
			for {
				t.Run()
				time.Sleep(t.SleepTime())
			}
		})
	}
}
