package tasks

import "time"

type Task interface {
	Run()

	SleepTime() time.Duration
}

func Register(t Task) {
	go func(t Task) {
		for {
			t.Run()
			time.Sleep(t.SleepTime())
		}
	}(t)
}
