package extraction

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestRunPool_BoundsConcurrency(t *testing.T) {
	l, _ := test.NewNullLogger()
	const N = 32
	const workers = 4

	var inflight int32
	var maxInflight int32

	jobs := make([]string, N)
	for i := range jobs {
		jobs[i] = "wz"
	}

	worker := func(ctx context.Context, _ string) error {
		cur := atomic.AddInt32(&inflight, 1)
		for {
			prev := atomic.LoadInt32(&maxInflight)
			if cur <= prev || atomic.CompareAndSwapInt32(&maxInflight, prev, cur) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&inflight, -1)
		return nil
	}

	runPool(context.Background(), l, jobs, workers, worker)

	if got := atomic.LoadInt32(&maxInflight); got > int32(workers) {
		t.Fatalf("maxInflight=%d exceeded workers=%d", got, workers)
	}
}

func TestRunPool_ContinuesOnError(t *testing.T) {
	l, _ := test.NewNullLogger()
	jobs := []string{"a", "b", "c"}
	var ran int32
	var mu sync.Mutex
	worker := func(ctx context.Context, j string) error {
		mu.Lock()
		atomic.AddInt32(&ran, 1)
		mu.Unlock()
		if j == "b" {
			return context.Canceled // stand-in for a per-unit error
		}
		return nil
	}
	runPool(context.Background(), l, jobs, 2, worker)
	if atomic.LoadInt32(&ran) != 3 {
		t.Fatalf("expected all 3 to run, ran=%d", ran)
	}
}
