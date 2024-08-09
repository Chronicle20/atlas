package tasks

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

type Task interface {
	Run()

	SleepTime() time.Duration
}

func Register(l logrus.FieldLogger, ctx context.Context) func(t Task) {
	return func(t Task) {
		go func(t Task) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				default:
					t.Run()
					time.Sleep(t.SleepTime())
				}
			}
		}(t)
	}
}
