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
			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				case <-time.After(t.SleepTime()):
					t.Run()
				}
			}
		}(t)
	}
}
