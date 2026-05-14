package extraction

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// runPool runs `worker` over each job in `jobs` with at most `workers` in
// flight. A per-job error does not abort sibling work — the worker logs the
// error and the pool continues. The function returns when all jobs complete or
// ctx is cancelled.
//
// This is the in-process fan-out path used by Extract (whole-list). The
// cross-pod path uses Kafka partition assignment instead.
func runPool[T any](ctx context.Context, l logrus.FieldLogger, jobs []T, workers int, worker func(context.Context, T) error) {
	if workers < 1 {
		workers = 1
	}
	ch := make(chan T)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			wl := l.WithField("worker", workerId)
			for j := range ch {
				if ctx.Err() != nil {
					return
				}
				if err := worker(ctx, j); err != nil {
					wl.WithError(err).Warn("pool worker returned error; continuing")
				}
			}
		}(i)
	}
	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		ch <- j
	}
	close(ch)
	wg.Wait()
}
